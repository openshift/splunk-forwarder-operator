package kube

import (
	"reflect"
	"testing"

	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

func TestGetVolumes(t *testing.T) {
	var hostPathDirectoryTypeForPtr = corev1.HostPathDirectory

	type args struct {
		mountHost    bool
		mountSecret  bool
		instanceName string
	}
	tests := []struct {
		name string
		args args
		want []corev1.Volume
	}{
		{
			name: "Host-true secret-false",
			args: args{
				mountHost:    true,
				mountSecret:  false,
				instanceName: "test",
			},
			want: []corev1.Volume{
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
					Name: "splunk-state",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/misc",
							Type: &hostPathDirectoryTypeForPtr,
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
				{
					Name: "test-internalsplunk",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-internalsplunk",
							},
						},
					},
				},
			},
		},
		{
			name: "Host-true secret-true",
			args: args{
				mountHost:    true,
				mountSecret:  true,
				instanceName: "test",
			},
			want: []corev1.Volume{
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
					Name: "splunk-state",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/misc",
							Type: &hostPathDirectoryTypeForPtr,
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
				{
					Name: config.SplunkAuthSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: config.SplunkAuthSecretName,
						},
					},
				},
			},
		},
		{
			name: "Host-false secret-false",
			args: args{
				mountHost:    false,
				mountSecret:  false,
				instanceName: "test",
			},
			want: []corev1.Volume{
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
					Name: "splunk-state",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/misc",
							Type: &hostPathDirectoryTypeForPtr,
						},
					},
				},
				{
					Name: "test-hfconfig",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-hfconfig",
							},
						},
					},
				},
				{
					Name: "test-internalsplunk",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-internalsplunk",
							},
						},
					},
				},
			},
		},
		{
			name: "Host-false secret-true",
			args: args{
				mountHost:    false,
				mountSecret:  true,
				instanceName: "test",
			},
			want: []corev1.Volume{
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
					Name: "splunk-state",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/misc",
							Type: &hostPathDirectoryTypeForPtr,
						},
					},
				},
				{
					Name: "test-hfconfig",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-hfconfig",
							},
						},
					},
				},
				{
					Name: config.SplunkAuthSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: config.SplunkAuthSecretName,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVolumes(tt.args.mountHost, tt.args.mountSecret, tt.args.instanceName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}
