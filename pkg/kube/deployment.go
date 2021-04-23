package kube

import (
	"strconv"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateDeployment returns a deployment that can be created with the oc client
func GenerateDeployment(instance *sfv1alpha1.SplunkForwarder) *appsv1.Deployment {
	var replicas int32 = instance.Spec.HeavyForwarderReplicas
	var selector string = instance.Spec.HeavyForwarderSelector
	if replicas == 0 {
		replicas = 2
	}
	var terminationGracePeriodSeconds int64 = 10

	var licenseAccepted string = "no"
	if instance.Spec.SplunkLicenseAccepted {
		licenseAccepted = "yes"
	}
	var envVars = []corev1.EnvVar{
		{
			Name:  "SPLUNK_ACCEPT_LICENSE",
			Value: licenseAccepted,
		},
	}

	var runAsUserID int64 = 1000
	podSecurityContext := corev1.PodSecurityContext{RunAsUser: &runAsUserID}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-deployment",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name,
			},
			Annotations: map[string]string{
				"genVersion": strconv.FormatInt(instance.Generation, 10),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "splunk-heavy-forwarder",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: "RollingUpdate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "splunk-heavy-forwarder",
					Namespace: instance.Namespace,
					Labels: map[string]string{
						"name": "splunk-heavy-forwarder",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/" + selector,
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,

					NodeSelector: map[string]string{
						"node-role.kubernetes.io": selector,
					},

					Containers: []corev1.Container{
						{
							Name:            "splunk-hf",
							ImagePullPolicy: corev1.PullAlways,
							// TEMPORARY: hardcode by-digest pull spec
							Image: "quay.io/app-sre/splunk-heavyforwarder@sha256:49b40c2c5d79913efb7eff9f3bf9c7348e322f619df10173e551b2596913d52a",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8089,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources:              corev1.ResourceRequirements{},
							TerminationMessagePath: "/dev/termination-log",

							Env: envVars,

							VolumeMounts: GetHeavyForwarderVolumeMounts(instance),
						},
					},
					Volumes:         GetVolumes(false, true, instance.Name),
					SecurityContext: &podSecurityContext,
				},
			},
		},
	}
}
