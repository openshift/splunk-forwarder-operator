package splunkforwarder

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/splunk-forwarder-operator/config"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_splunkforwarder")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SplunkForwarder Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSplunkForwarder{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("splunkforwarder-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SplunkForwarder
	err = c.Watch(&source.Kind{Type: &sfv1alpha1.SplunkForwarder{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSplunkForwarder implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSplunkForwarder{}

// ReconcileSplunkForwarder reconciles a SplunkForwarder object
type ReconcileSplunkForwarder struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	reqLogger logr.Logger
}

// CheckGenerationVersionOlder is a function that checks against an annontations map with a splunk forwarder instance to compare the saved
// generation to the CR generation
func (r *ReconcileSplunkForwarder) CheckGenerationVersionOlder(annontations map[string]string, instance *sfv1alpha1.SplunkForwarder) bool {
	if annontations["genVersion"] == "" {
		r.reqLogger.Info("genVersion missing")
		return true
	}

	genVersion, err := strconv.ParseInt(annontations["genVersion"], 10, 64)
	if err != nil {
		r.reqLogger.Info("Error parsing genVersion")
		return true
	}

	if genVersion < instance.Generation {
		return true
	}

	return false
}

// Reconcile reads that state of the cluster for a SplunkForwarder object and makes changes based on the state read
// and what is in the SplunkForwarder.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSplunkForwarder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.reqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.reqLogger.Info("Reconciling SplunkForwarder")

	// Fetch the SplunkForwarder instance
	instance := &sfv1alpha1.SplunkForwarder{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// See if our Secret exists
	secFound := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: config.SplunkAuthSecretName, Namespace: request.Namespace}, secFound)
	if err != nil {
		return reconcile.Result{}, err
	}

	var clusterid string
	if instance.Spec.ClusterID != "" {
		clusterid = instance.Spec.ClusterID
	} else {
		configFound := &configv1.Infrastructure{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, configFound)
		if err != nil {
			r.reqLogger.Info(err.Error())
			clusterid = "openshift"
		} else {
			clusterid = configFound.Status.InfrastructureName
		}
	}

	// ConfigMaps
	// Define a new ConfigMap object
	// TODO(wshearn) - check instance.Spec.ClusterID, if it is empty look it up on the cluster.
	configMaps := kube.GenerateConfigMaps(instance, request.NamespacedName, clusterid)
	if instance.Spec.UseHeavyForwarder {
		configMaps = append(configMaps, kube.GenerateInternalConfigMap(instance, request.NamespacedName))
		configMaps = append(configMaps, kube.GenerateFilteringConfigMap(instance, request.NamespacedName))

	}

	for _, configmap := range configMaps {
		// Set SplunkForwarder instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, configmap, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this ConfigMap already exists
		cmFound := &corev1.ConfigMap{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: configmap.Name, Namespace: configmap.Namespace}, cmFound)
		if err != nil && errors.IsNotFound(err) {
			r.reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configmap.Namespace, "ConfigMap.Name", configmap.Name)
			err = r.client.Create(context.TODO(), configmap)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(cmFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(cmFound.GetAnnotations(), instance) {
			r.reqLogger.Info("Updating ConfigMap", "ConfigMap.Namespace", configmap.Namespace, "ConfigMap.Name", configmap.Name)
			err = r.client.Update(context.TODO(), configmap)
			if err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// DaemonSet
	daemonSet := kube.GenerateDaemonSet(instance)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, daemonSet, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DaemonSet already exists
	dsFound := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, dsFound)
	if err != nil && errors.IsNotFound(err) {
		r.reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.client.Create(context.TODO(), daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else if instance.CreationTimestamp.After(dsFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(dsFound.GetAnnotations(), instance) {
		err = r.client.Update(context.TODO(), daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Requeue to create the daemonset
		return reconcile.Result{Requeue: true}, nil
	}

	// Deployment
	deployment := kube.GenerateDeployment(instance)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	deploymentFound := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
	if instance.Spec.UseHeavyForwarder {
		if err != nil && errors.IsNotFound(err) {
			r.reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			err = r.client.Create(context.TODO(), deployment)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(deploymentFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(deploymentFound.GetAnnotations(), instance) {
			err = r.client.Delete(context.TODO(), deploymentFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the deployment
			return reconcile.Result{Requeue: true}, nil
		}
	} else { // The CR changed to not use the HF, so clean up the old deployment
		if err == nil {
			r.reqLogger.Info("Deleting the Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			err = r.client.Delete(context.TODO(), deploymentFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the deployment
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// Service
	service := kube.GenerateService(instance)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	serviceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
	if instance.Spec.UseHeavyForwarder {
		if err != nil && errors.IsNotFound(err) {
			r.reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.client.Create(context.TODO(), service)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(serviceFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(serviceFound.GetAnnotations(), instance) {
			err = r.client.Delete(context.TODO(), serviceFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the service
			return reconcile.Result{Requeue: true}, nil
		}
	} else { // The CR changed to not use the HF, so clean up the old service
		if err == nil {
			r.reqLogger.Info("Deleting the Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.client.Delete(context.TODO(), serviceFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the service
			return reconcile.Result{Requeue: true}, nil
		}
	}
	return reconcile.Result{}, nil
}
