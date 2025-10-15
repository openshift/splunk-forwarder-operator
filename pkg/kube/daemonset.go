package kube

import (
	"strconv"

	configv1 "github.com/openshift/api/config/v1"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func forwarderPullSpec(instance *sfv1alpha1.SplunkForwarder) string {
	var sep, suffix string
	// ImageDigest takes precedence.
	if instance.Spec.ImageDigest != "" {
		sep = "@"
		suffix = instance.Spec.ImageDigest
	} else {
		sep = ":"
		suffix = instance.Spec.ImageTag
	}
	return instance.Spec.Image + sep + suffix
}

// GenerateDaemonSet returns a daemonset that can be created with the oc client
func GenerateDaemonSet(instance *sfv1alpha1.SplunkForwarder, useHECToken bool, proxyConfig *configv1.Proxy) *appsv1.DaemonSet {

	var (
		runAsUID                      int64 = 0
		terminationGracePeriodSeconds int64 = 10
		priority                      int32 = 2000001000
	)
	isPrivContainer := true
	licenseAccepted := "no"
	if instance.Spec.SplunkLicenseAccepted {
		licenseAccepted = "yes"
	}
	envVars := []corev1.EnvVar{
		{
			Name:  "SPLUNK_ACCEPT_LICENSE",
			Value: licenseAccepted,
		},
		{
			Name: "HOSTNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}

	volumes := GetVolumes(true, true, useHECToken, instance.Name)

	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-ds",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name,
			},
			Annotations: map[string]string{
				"genVersion": strconv.FormatInt(instance.Generation, 10),
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
					PriorityClassName: "system-node-critical",
					Priority:          &priority,
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
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
							Image:           forwarderPullSpec(instance),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8089,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources:              corev1.ResourceRequirements{},
							TerminationMessagePath: "/dev/termination-log",

							Env: envVars,

							VolumeMounts: GetVolumeMounts(instance, useHECToken, proxyConfig),

							SecurityContext: &corev1.SecurityContext{
								Privileged: &isPrivContainer,
								RunAsUser:  &runAsUID,
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if useHECToken {
		daemonset.Spec.Template.Spec.InitContainers = []corev1.Container{
			getInitContainer(),
		}
	}

	return daemonset
}

func getInitContainer() corev1.Container {
	initScript := `cp /tmp/splunk-hec-token/outputs.conf /tmp/splunk-config/outputs.conf
chown 1000:1000 /tmp/splunk-config/outputs.conf`

	initContainer := corev1.Container{
		Name:  "init-config",
		Image: "image-registry.openshift-image-registry.svc:5000/openshift/cli:latest",
		Command: []string{
			"/bin/bash",
			"-c",
			initScript,
		},
		VolumeMounts: getInitVolumeMounts(),
	}
	return initContainer
}
