package kube

import (
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expectedDeployment(instance *sfv1alpha1.SplunkForwarder) *appsv1.Deployment {
	var expectedReplicas int32 = 2
	var expectedTerminationGracePeriodSeconds int64 = 10
	var expectedRunAsUserID int64 = 1000

	var hsfImage string
	if instance.Spec.HeavyForwarderDigest == "" {
		hsfImage = hfImage + ":" + imageTag
	} else {
		hsfImage = hfImage + "@" + heavyDigest
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "openshift-test",
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"genVersion": "10",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &expectedReplicas,
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
					Namespace: "openshift-test",
					Labels: map[string]string{
						"name": "splunk-heavy-forwarder",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/infra",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					TerminationGracePeriodSeconds: &expectedTerminationGracePeriodSeconds,

					NodeSelector: map[string]string{
						"node-role.kubernetes.io": "infra",
					},

					Containers: []corev1.Container{
						{
							Name:            "splunk-hf",
							ImagePullPolicy: corev1.PullAlways,
							Image:           hsfImage,
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

							VolumeMounts: GetHeavyForwarderVolumeMounts(instance),
						},
					},
					Volumes: GetVolumes(false, true, "test"),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &expectedRunAsUserID,
					},
				},
			},
		},
	}
}

func TestGenerateDeployment(t *testing.T) {

	tests := []struct {
		name     string
		instance *sfv1alpha1.SplunkForwarder
	}{
		// NOTE: These tests only make sense with useHeavy==true
		// TODO: Permutations with useTag==useHFDigest==false should be invalid and produce a
		// predictable error.
		{
			name:     "With digest",
			instance: splunkForwarderInstance(true, false, false, true),
		},
		{
			name:     "With HF digest; image digest is moot",
			instance: splunkForwarderInstance(true, false, true, true),
		},
		{
			name:     "Heavy Forwarder deployment with tag",
			instance: splunkForwarderInstance(true, true, false, false),
		},
		{
			name:     "Digest overrides tag",
			instance: splunkForwarderInstance(true, true, false, true),
		},
		{
			name:     "With HF tag; image digest is moot",
			instance: splunkForwarderInstance(true, true, true, false),
		},
		{
			name:     "HF digest overrides tag; image digest is moot",
			instance: splunkForwarderInstance(true, true, true, true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeepEqualWithDiff(t,
				expectedDeployment(tt.instance),
				GenerateDeployment(tt.instance))
		})
	}
}
