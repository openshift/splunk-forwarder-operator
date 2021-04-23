package kube

import (
	"reflect"
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	instanceName      = "test"
	instanceNamespace = "openshift-test"
	image             = "test-image"
	imageTag          = "0.0.1"
)

func TestGenerateDaemonSet(t *testing.T) {
	var expectedRunAsUID int64 = 0
	var expectedIsPrivContainer bool = true
	var expectedTerminationGracePeriodSeconds int64 = 10
	var testHFInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			UseHeavyForwarder:      true,
			HeavyForwarderReplicas: 0,
			SplunkLicenseAccepted:  true,
			Image:                  image,
			ImageTag:               imageTag,
		},
	}
	var testNoHFInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			UseHeavyForwarder:      false,
			HeavyForwarderReplicas: 0,
			SplunkLicenseAccepted:  true,
			Image:                  image,
			ImageTag:               imageTag,
		},
	}

	type args struct {
		instance *sfv1alpha1.SplunkForwarder
	}
	tests := []struct {
		name string
		args args
		want *appsv1.DaemonSet
	}{
		{
			name: "Test Daemonset with HF",
			args: args{
				instance: testHFInstance,
			},
			want: &appsv1.DaemonSet{
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
									Image:           image + ":" + imageTag,
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

									VolumeMounts: GetVolumeMounts(testHFInstance),

									SecurityContext: &corev1.SecurityContext{
										Privileged: &expectedIsPrivContainer,
										RunAsUser:  &expectedRunAsUID,
									},
								},
							},
							Volumes: GetVolumes(true, false, instanceName),
						},
					},
				},
			},
		},
		{
			name: "Test Daemonset without HF",
			args: args{
				instance: testNoHFInstance,
			},
			want: &appsv1.DaemonSet{
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
									Image:           image + ":" + imageTag,
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

									VolumeMounts: GetVolumeMounts(testNoHFInstance),

									SecurityContext: &corev1.SecurityContext{
										Privileged: &expectedIsPrivContainer,
										RunAsUser:  &expectedRunAsUID,
									},
								},
							},
							Volumes: GetVolumes(true, true, instanceName),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateDaemonSet(tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateDaemonSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
