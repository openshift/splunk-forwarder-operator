package secret

import (
	"context"
	goerr "errors"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	"github.com/openshift/splunk-forwarder-operator/pkg/kube"
)

var log = logf.Log.WithName("controller_secret")

// mySecretPredicate filters out any events not related to our Secret.
func mySecretPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return passes(e.Object) },
		DeleteFunc: func(e event.DeleteEvent) bool { return e.Object.GetName() == config.SplunkHECTokenSecretName },
		UpdateFunc: func(e event.UpdateEvent) bool {
			return dataChanged(e.ObjectOld.(*corev1.Secret), e.ObjectNew.(*corev1.Secret))
		},
		GenericFunc: func(e event.GenericEvent) bool { return passes(e.Object) },
	}
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
	return s.GetName() == config.SplunkAuthSecretName || s.GetName() == config.SplunkHECTokenSecretName
}

func dataChanged(old, new *corev1.Secret) bool {
	return !reflect.DeepEqual(old.Data, new.Data)
}

// blank assignment to verify that SecretReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &SecretReconciler{}

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
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
	err := r.Client.List(ctx, sfCrds, listOpts...)
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

	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: config.SplunkHECTokenSecretName, Namespace: request.Namespace}, secret)
	if errors.IsNotFound(err) {
		reqLogger.Info("HEC Token secret not found, falling back to legacy mTLS authentication")
		err = r.Client.Get(ctx, types.NamespacedName{Namespace: request.Namespace, Name: config.SplunkAuthSecretName}, secret)
		if errors.IsNotFound(err) {
			reqLogger.Info("No Splunk auth secrets found, not restarting DaemonSet")
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		reqLogger.Info("Using HEC Token for Splunk authentication")
	}

	currentDaemonSet := &appsv1.DaemonSet{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: sfCrd.Name + "-ds", Namespace: request.Namespace}, currentDaemonSet)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	hecSecretPresent := secret.Name == config.SplunkHECTokenSecretName
	newDaemonSet := kube.GenerateDaemonSet(sfCrd, hecSecretPresent)
	if err := controllerutil.SetControllerReference(sfCrd, newDaemonSet, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = r.Client.Delete(ctx, currentDaemonSet)
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", newDaemonSet.Namespace, "DaemonSet.Name", newDaemonSet.Name)
	err = r.Client.Create(ctx, newDaemonSet)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(mySecretPredicate()).
		Complete(r)
}
