package secret

import (
	"context"
	goerr "errors"
	ctrl "sigs.k8s.io/controller-runtime"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	"github.com/openshift/splunk-forwarder-operator/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &SecretReconciler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// mySecretPredicate filters out any events not related to our Secret.
var mySecretPredicate = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool { return passes(e.Object) },
	DeleteFunc: func(e event.DeleteEvent) bool { return passes(e.Object) },
	// UpdateFunc passes if *either* the new or old object is one we care about.
	UpdateFunc: func(e event.UpdateEvent) bool {
		return passes(e.ObjectOld) || passes(e.ObjectNew)
	},
	GenericFunc: func(e event.GenericEvent) bool { return passes(e.Object) },
}

func passes(o runtime.Object) bool {
	if o == nil {
		log.Error(nil, "No Object for event!")
		return false
	}
	s, ok := o.(*corev1.Secret)
	if !ok {
		log.Error(nil, "Not a Secret (this should never happen)")
		return false
	}
	return s.GetName() == config.SplunkAuthSecretName
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, mySecretPredicate)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that SecretReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &SecretReconciler{}

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *SecretReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// TODO: Fix this namespace, look for our crd and check against the namespace it lives in

	sfCrds := &sfv1alpha1.SplunkForwarderList{}
	listOpts := []client.ListOption{
		client.InNamespace(request.Namespace),
	}
	err := r.client.List(context.TODO(), sfCrds, listOpts...)
	// Error getting CR
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(sfCrds.Items) > 1 {
		return reconcile.Result{}, goerr.New("More than one CR in namespace")
	}

	// Our CR does not exist in this namespace, just ignore and continue
	if len(sfCrds.Items) != 1 {
		return reconcile.Result{}, nil
	}
	sfCrd := &sfCrds.Items[0]

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the Secret instance
	instance := &corev1.Secret{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Error(err, "Splunk Auth Secret was deleted, recreate it or delete the CRD, not restarting DaemonSet")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	daemonSet := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: sfCrd.Name + "-ds", Namespace: instance.Namespace}, daemonSet)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// We don't need to do anyhting if the DaemonSet was Created after the Secret
	if daemonSet.CreationTimestamp.After(instance.CreationTimestamp.Time) {
		return reconcile.Result{}, nil
	}

	err = r.client.Delete(context.TODO(), daemonSet)
	if err != nil {
		return reconcile.Result{}, err
	}

	// DaemonSet
	daemonSet = kube.GenerateDaemonSet(sfCrd)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(sfCrd, daemonSet, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DaemonSet already exists
	dsFound := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, dsFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.client.Create(context.TODO(), daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
