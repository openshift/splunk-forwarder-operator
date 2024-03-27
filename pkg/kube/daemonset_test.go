package kube

import (
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// daemonSetInstance produces (a pointer to) an expected DaemonSet produced by GenerateDaemonSet.
// Parameters;
// - sfInstance: SplunkForwarder instance under test.
func expectedDaemonSet(instance *sfv1alpha1.SplunkForwarder) *appsv1.DaemonSet {
	var expectedRunAsUID int64 = 0
	var expectedIsPrivContainer bool = true
	var expectedTerminationGracePeriodSeconds int64 = 10

	useVolumeSecret := !instance.Spec.UseHeavyForwarder
	var sfImage string
	if instance.Spec.ImageDigest == "" {
		sfImage = image + ":" + imageTag
	} else {
		sfImage = image + "@" + imageDigest
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-ds",
			Namespace: instanceNamespace,
			Labels: map[string]string{
				"app": instanceName,
			},
			Annotations: map[string]string{
				"genVersion": "10",
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
					Namespace: instanceNamespace,
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
					TerminationGracePeriodSeconds: &expectedTerminationGracePeriodSeconds,

					Containers: []corev1.Container{
						{
							Name:            "splunk-uf",
							ImagePullPolicy: corev1.PullAlways,
							Image:           sfImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8089,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources:              corev1.ResourceRequirements{},
							TerminationMessagePath: "/dev/termination-log",
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: config.ProxyConfigMapName,
										},
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SPLUNK_ACCEPT_LICENSE",
									Value: "yes",
								},
								{
									Name: "HOSTNAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},

							VolumeMounts: GetVolumeMounts(instance),

							SecurityContext: &corev1.SecurityContext{
								Privileged: &expectedIsPrivContainer,
								RunAsUser:  &expectedRunAsUID,
							},
						},
					},
					Volumes: GetVolumes(true, useVolumeSecret, false, instanceName),
				},
			},
		},
	}
}

func TestGenerateDaemonSet(t *testing.T) {
	tests := []struct {
		name     string
		instance *sfv1alpha1.SplunkForwarder
	}{
		// TODO: The following configurations should be invalid and produce a predictable error:
		// - splunkForwarderInstance(any, false, false, any)
		//   (Can't make sf pull spec when neither tag nor digest is present.)
		{
			name:     "Without HF, with image digest",
			instance: splunkForwarderInstance(false, false, true, false),
		},
		{
			name: "Without HF, with digests",
			// The HF digest is ignored because not using HF
			instance: splunkForwarderInstance(false, false, true, true),
		},
		{
			name:     "Test Daemonset without HF, with tags",
			instance: splunkForwarderInstance(false, true, false, false),
		},
		{
			name: "Test Daemonset without HF, with tags and moot HF digest",
			// The HF digest is ignored because not using HF
			instance: splunkForwarderInstance(false, true, false, true),
		},
		{
			name:     "Without HF, digest overrides tag",
			instance: splunkForwarderInstance(false, true, true, false),
		},
		{
			name: "Without HF, digest overrides tag, moot HF digest",
			// The HF digest is ignored because not using HF
			instance: splunkForwarderInstance(false, true, true, true),
		},
		{
			name: "With HF, with image digest",
			// NOTE: useHeavy && !useTag && !useHeavyDigest should be an invalid configuration,
			// but GenerateDaemonSet won't catch that.
			instance: splunkForwarderInstance(true, false, true, false),
		},
		{
			name:     "With HF, with digests",
			instance: splunkForwarderInstance(true, false, true, true),
		},
		{
			name:     "Test Daemonset, with HF, with tags",
			instance: splunkForwarderInstance(true, true, false, false),
		},
		{
			name: "With HF, image tag and HF digest",
			// IRL this will produce a DaemonSet with the sf image by tag, and a Deployment with
			// the shf image by digest.
			instance: splunkForwarderInstance(true, true, false, true),
		},
		{
			name: "With HF, image digest overrides tag",
			// IRL this will produce a DaemonSet with the sf image by digest, and a Deployment with
			// the shf image by tag.
			instance: splunkForwarderInstance(true, true, true, false),
		},
		{
			name: "With HF, digests overrides tags",
			// IRL this will produce a DaemonSet with the sf image by digest, and a Deployment with
			// the shf image also by digest.
			instance: splunkForwarderInstance(true, true, true, true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeepEqualWithDiff(t,
				expectedDaemonSet(tt.instance),
				GenerateDaemonSet(tt.instance, false))
		})
	}
}
