package splunkforwarder

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	"github.com/openshift/splunk-forwarder-operator/pkg/kube"
)

var (
	log = logf.Log.WithName("controller_splunkforwarder")
)

// SplunkForwarderReconciler reconciles a SplunkForwarder object
type SplunkForwarderReconciler struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	ReqLogger logr.Logger
}

// CheckGenerationVersionOlder is a function that checks against an annontations map with a splunk forwarder instance to compare the saved
// generation to the CR generation
func (r *SplunkForwarderReconciler) CheckGenerationVersionOlder(annontations map[string]string, instance *sfv1alpha1.SplunkForwarder) bool {
	if annontations["genVersion"] == "" {
		r.ReqLogger.Info("genVersion missing")
		return true
	}

	genVersion, err := strconv.ParseInt(annontations["genVersion"], 10, 64)
	if err != nil {
		r.ReqLogger.Info("Error parsing genVersion")
		return true
	}

	if genVersion < instance.Generation {
		return true
	}

	return false
}

//+kubebuilder:rbac:groups=splunkforwarder.managed.openshift.io,resources=splunkforwarders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=splunkforwarder.managed.openshift.io,resources=splunkforwarders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=splunkforwarder.managed.openshift.io,resources=splunkforwarders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SplunkForwarderReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {

	r.ReqLogger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.ReqLogger.Info("Reconciling SplunkForwarder")

	// Fetch the SplunkForwarder instance
	instance := &sfv1alpha1.SplunkForwarder{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
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
	err = r.Client.Get(ctx, types.NamespacedName{Name: config.SplunkAuthSecretName, Namespace: request.Namespace}, secFound)
	if err != nil {
		return reconcile.Result{}, err
	}

	var clusterid string
	if instance.Spec.ClusterID != "" {
		clusterid = instance.Spec.ClusterID
	} else {
		configFound := &configv1.Infrastructure{}
		err = r.Client.Get(ctx, types.NamespacedName{Name: "cluster"}, configFound)
		if err != nil {
			r.ReqLogger.Info(err.Error())
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
		if err := controllerutil.SetControllerReference(instance, configmap, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this ConfigMap already exists
		cmFound := &corev1.ConfigMap{}
		err = r.Client.Get(ctx, types.NamespacedName{Name: configmap.Name, Namespace: configmap.Namespace}, cmFound)
		if err != nil && errors.IsNotFound(err) {
			r.ReqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configmap.Namespace, "ConfigMap.Name", configmap.Name)
			err = r.Client.Create(ctx, configmap)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(cmFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(cmFound.GetAnnotations(), instance) {
			r.ReqLogger.Info("Updating ConfigMap", "ConfigMap.Namespace", configmap.Namespace, "ConfigMap.Name", configmap.Name)
			err = r.Client.Update(ctx, configmap)
			if err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	}

	proxy := configv1.Proxy{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "cluster"}, &proxy)
	if err != nil {
		r.ReqLogger.Info("Unable to determine proxy status, assuming defaults", "Error", err)
	}

	// DaemonSet
	daemonSet := kube.GenerateDaemonSet(instance, proxy.Status.HTTPProxy != proxy.Status.HTTPSProxy)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, daemonSet, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DaemonSet already exists
	dsFound := &appsv1.DaemonSet{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, dsFound)
	if err != nil && errors.IsNotFound(err) {
		r.ReqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.Client.Create(ctx, daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else if instance.CreationTimestamp.After(dsFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(dsFound.GetAnnotations(), instance) {
		err = r.Client.Delete(ctx, daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Requeue to create the daemonset
		return reconcile.Result{Requeue: true}, nil
	}

	// Deployment
	deployment := kube.GenerateDeployment(instance)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	deploymentFound := &appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deploymentFound)
	if instance.Spec.UseHeavyForwarder {
		if err != nil && errors.IsNotFound(err) {
			r.ReqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			err = r.Client.Create(ctx, deployment)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(deploymentFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(deploymentFound.GetAnnotations(), instance) {
			err = r.Client.Delete(ctx, deploymentFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the deployment
			return reconcile.Result{Requeue: true}, nil
		}
	} else { // The CR changed to not use the HF, so clean up the old deployment
		if err == nil {
			r.ReqLogger.Info("Deleting the Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			err = r.Client.Delete(ctx, deploymentFound)
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
	if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	serviceFound := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)
	if instance.Spec.UseHeavyForwarder {
		if err != nil && errors.IsNotFound(err) {
			r.ReqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.Client.Create(ctx, service)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else if instance.CreationTimestamp.After(serviceFound.CreationTimestamp.Time) || r.CheckGenerationVersionOlder(serviceFound.GetAnnotations(), instance) {
			err = r.Client.Delete(ctx, serviceFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the service
			return reconcile.Result{Requeue: true}, nil
		}
	} else { // The CR changed to not use the HF, so clean up the old service
		if err == nil {
			r.ReqLogger.Info("Deleting the Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			err = r.Client.Delete(ctx, serviceFound)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Requeue to create the service
			return reconcile.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SplunkForwarderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sfv1alpha1.SplunkForwarder{}).
		Complete(r)
}
