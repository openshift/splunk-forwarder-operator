package kube

import (
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
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
	var expectedPriorityClassName string = "system-node-critical"
	var expectedPriority int32 = 2000001000

	useVolumeSecret := true
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
					PriorityClassName: expectedPriorityClassName,
					Priority:          &expectedPriority,
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
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

							VolumeMounts: GetVolumeMounts(instance, false),

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
		name        string
		instance    *sfv1alpha1.SplunkForwarder
		useHECToken bool
	}{
		// TODO: The following configurations should be invalid and produce a predictable error:
		// - splunkForwarderInstance(false, false)
		//   (Can't make sf pull spec when neither tag nor digest is present.)
		{
			name:     "Test Daemonset with image digest",
			instance: splunkForwarderInstance(true),
		},
		{
			name:     "Test Daemonset with tags",
			instance: splunkForwarderInstance(false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := expectedDaemonSet(tt.instance)
			actual := GenerateDaemonSet(tt.instance, tt.useHECToken)
			DeepEqualWithDiff(t, expected, actual)
		})
	}
}
