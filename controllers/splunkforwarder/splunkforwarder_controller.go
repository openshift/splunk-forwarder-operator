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

	// Get cluster proxy config here
	proxyConfig := &configv1.Proxy{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "cluster"}, proxyConfig)
	if !errors.IsNotFound(err) || (proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "") {
		proxyConfig = nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// ConfigMaps
	// Define a new ConfigMap object
	// TODO(wshearn) - check instance.Spec.ClusterID, if it is empty look it up on the cluster.
	configMaps := kube.GenerateConfigMaps(instance, request.NamespacedName, clusterid, proxyConfig)

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

	useHECToken := false
	hecToken := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: config.SplunkHECTokenSecretName, Namespace: request.Namespace}, hecToken)
	if errors.IsNotFound(err) {
		r.ReqLogger.Info("HTTP Event Collector token not present, using mTLS authentication")
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		r.ReqLogger.Info("HTTP Event Collector token found, using HEC mode for Splunk Universal Forwarder")
		useHECToken = true
	}

	// DaemonSet
	daemonSet := kube.GenerateDaemonSet(instance, useHECToken, proxyConfig)
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

	// Service
	service := kube.GenerateService(instance)
	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	serviceFound := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, serviceFound)

	if err == nil {
		r.ReqLogger.Info("Deleting the Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Delete(ctx, serviceFound)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Requeue to create the service
		return reconcile.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SplunkForwarderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sfv1alpha1.SplunkForwarder{}).
		Complete(r)
}
