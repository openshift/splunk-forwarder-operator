package kube

import (
	"reflect"
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateDeployment(t *testing.T) {
	var expectedReplicas int32 = 2
	var expectedTerminationGracePeriodSeconds int64 = 10
	var expectedRunAsUserID int64 = 1000
	var testInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Namespace:  "openshift-test",
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			UseHeavyForwarder:      true,
			HeavyForwarderReplicas: 0,
			SplunkLicenseAccepted:  true,
			HeavyForwarderSelector: "infra",
			HeavyForwarderImage:    "test-image",
			ImageTag:               "0.0.1",
		},
	}

	type args struct {
		instance *sfv1alpha1.SplunkForwarder
	}

	tests := []struct {
		name string
		args args
		want *appsv1.Deployment
	}{
		{
			name: "Heavy Forwarder deployment",
			args: args{
				instance: testInstance,
			},
			want: &appsv1.Deployment{
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
									Image:           "test-image:0.0.1",
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

									VolumeMounts: GetHeavyForwarderVolumeMounts(testInstance),
								},
							},
							Volumes: GetVolumes(false, true, "test"),
							SecurityContext: &corev1.PodSecurityContext{
								RunAsUser: &expectedRunAsUserID,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateDeployment(tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}
