package kube

import (
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var envVars = []corev1.EnvVar{
	{
		Name:  "SPLUNK_ACCEPT_LICENSE",
		Value: "yes",
	},
}

var hostPathDirectoryTypeForPtr = corev1.HostPathDirectory

// GenerateDaemonSet returns a daemonset that can be created with the oc client
func GenerateDaemonSet(instance *sfv1alpha1.SplunkForwarder) *appsv1.DaemonSet {

	var runAsUID int64 = 0
	var isPrivContainer bool = true
	var terminationGracePeriodSeconds int64 = 10

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-ds",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name,
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
					Namespace: instance.Namespace,
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
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,

					Containers: []corev1.Container{
						{
							Name:            "splunk-uf",
							ImagePullPolicy: corev1.PullAlways,
							Image:           instance.Spec.Image + ":" + instance.Spec.ImageVersion,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8089,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources:              corev1.ResourceRequirements{},
							TerminationMessagePath: "/dev/termination-log",

							Env: envVars,

							VolumeMounts: GetVolumeMounts(),

							SecurityContext: &corev1.SecurityContext{
								Privileged: &isPrivContainer,
								RunAsUser:  &runAsUID,
							},
						},
					},
					Volumes: GetVolumes(),
				},
			},
		},
	}
}
