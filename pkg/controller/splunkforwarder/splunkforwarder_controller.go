package splunkforwarder

import (
	"context"

	splunkforwarderv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
	err = c.Watch(&source.Kind{Type: &splunkforwarderv1alpha1.SplunkForwarder{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner SplunkForwarder
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &splunkforwarderv1alpha1.SplunkForwarder{},
	})
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
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SplunkForwarder object and makes changes based on the state read
// and what is in the SplunkForwarder.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSplunkForwarder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SplunkForwarder")

	// Fetch the SplunkForwarder instance
	instance := &splunkforwarderv1alpha1.SplunkForwarder{}
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

	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set SplunkForwarder instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

func newDaemonSetForCR(cr *splunkforwarderv1alpha1.SplunkForwarder) *appsv1.DaemonSet {
	boolTrueForPtr := true
	var hostPathDirectoryTypeForPtr corev1.HostPathType
	hostPathDirectoryTypeForPtr = corev1.HostPathDirectory
	var int0ForPtr int64
	int0ForPtr = 0

	var int10ForPtr int64
	int10ForPtr = 10

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-ds",
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": cr.Name,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "splunk-forwarder",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "splunk-forwarder",
					Namespace: cr.Namespace,
					Labels: map[string]string{
						"name": "splunk-forwarder",
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"beta.kubernetes.io/os": "linux",
					},

					ServiceAccountName: "default",
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
					},
					TerminationGracePeriodSeconds: &int10ForPtr,

					Containers: []corev1.Container{
						{
							Name:            "splunk-uf",
							ImagePullPolicy: corev1.PullAlways,
							Image:           cr.Spec.Image + ":" + cr.Spec.ImageVersion,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8089,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources:              corev1.ResourceRequirements{},
							TerminationMessagePath: "/dev/termination-log",

							Env: []corev1.EnvVar{
								{
									Name:  "SPLUNK_ACCEPT_LICENSE",
									Value: "yes",
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								// Splunk Forwarder Certificate Mounts
								{
									Name:      "splunk-auth-default",
									MountPath: "/opt/splunkforwarder/etc/splunk/apps/splunkauth/default",
								},
								{
									Name:      "splunk-auth-local",
									MountPath: "/opt/splunkforwarder/etc/splunk/apps/splunkauth/local",
								},
								{
									Name:      "splunk-auth-metadata",
									MountPath: "/opt/splunkforwarder/etc/splunk/apps/splunkauth/metadata",
								},

								// Inputs Mount
								{
									Name:      "osd-monitored-logs-local",
									MountPath: "/opt/splunkforwarder/etc/splunk/apps/osd_monitored_logs/local",
								},
								{
									Name:      "osd-monitored-logs-metadata",
									MountPath: "/opt/splunkforwarder/etc/splunk/apps/osd_monitored_logs/metadata",
								},

								// Host Mount
								{
									Name:      "host",
									MountPath: "/host",
									ReadOnly:  true,
								},
							},

							SecurityContext: &corev1.SecurityContext{
								Privileged: &boolTrueForPtr,
								RunAsUser:  &int0ForPtr,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "splunk-auth-default",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "splunk-auth-default",
								},
							},
						},
						{
							Name: "splunk-auth-local",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "splunk-auth-local",
								},
							},
						},
						{
							Name: "splunk-auth-metadata",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "splunk-auth-metadata",
								},
							},
						},

						{
							Name: "osd-monitored-logs-local",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "osd-monitored-logs-local",
									},
								},
							},
						},
						{
							Name: "osd-monitored-logs-metadata",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "osd-monitored-logs-metadata",
									},
								},
							},
						},

						{
							Name: "host",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
									Type: &hostPathDirectoryTypeForPtr,
								},
							},
						},
					},
				},
			},
		},
	}
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *splunkforwarderv1alpha1.SplunkForwarder) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
