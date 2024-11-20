package kube

import (
	"reflect"
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetVolumeMounts(t *testing.T) {
	type args struct {
		instance    *sfv1alpha1.SplunkForwarder
		useHECToken bool
	}
	var mountPropagationMode = corev1.MountPropagationHostToContainer
	tests := []struct {
		name string
		args args
		want []corev1.VolumeMount
	}{
		{
			name: "Using Heavy forwarder",
			args: args{
				instance: &sfv1alpha1.SplunkForwarder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: sfv1alpha1.SplunkForwarderSpec{
						UseHeavyForwarder: true,
					},
				},
			},
			want: []corev1.VolumeMount{
				{
					Name:      "test-internalsplunk",
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/default",
				},
				{
					Name:      "test-internalsplunk",
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/local",
				},
				{
					Name:      "test-internalsplunk",
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/metadata",
				},
				{
					Name:      "osd-monitored-logs-local",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
				},
				{
					Name:      "osd-monitored-logs-metadata",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
				},
				{
					Name:      "splunk-state",
					MountPath: "/opt/splunkforwarder/var/lib",
				},
				{
					Name:             "host",
					MountPath:        "/host",
					MountPropagation: &mountPropagationMode,
					ReadOnly:         true,
				},
			},
		},
		{
			name: "Don't use heaver forwarder",
			args: args{
				instance: &sfv1alpha1.SplunkForwarder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: sfv1alpha1.SplunkForwarderSpec{
						UseHeavyForwarder: false,
					},
				},
			},
			want: []corev1.VolumeMount{
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/default",
				},
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/local",
				},
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/metadata",
				},
				{
					Name:      "osd-monitored-logs-local",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
				},
				{
					Name:      "osd-monitored-logs-metadata",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
				},
				{
					Name:      "splunk-state",
					MountPath: "/opt/splunkforwarder/var/lib",
				},
				{
					Name:             "host",
					MountPath:        "/host",
					MountPropagation: &mountPropagationMode,
					ReadOnly:         true,
				},
			},
		},
		{
			name: "Use HEC Token config",
			args: args{
				instance: &sfv1alpha1.SplunkForwarder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: sfv1alpha1.SplunkForwarderSpec{
						UseHeavyForwarder: false,
					},
				},
				useHECToken: true,
			},
			want: []corev1.VolumeMount{
				{
					Name:      "splunk-config",
					MountPath: "/opt/splunkforwarder/etc/system/local",
				},
				{
					Name:      "osd-monitored-logs-local",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
				},
				{
					Name:      "osd-monitored-logs-metadata",
					MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
				},
				{
					Name:      "splunk-state",
					MountPath: "/opt/splunkforwarder/var/lib",
				},
				{
					Name:             "host",
					MountPath:        "/host",
					MountPropagation: &mountPropagationMode,
					ReadOnly:         true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVolumeMounts(tt.args.instance, tt.args.useHECToken); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVolumeMounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHeavyForwarderVolumeMounts(t *testing.T) {
	type args struct {
		instance *sfv1alpha1.SplunkForwarder
	}
	tests := []struct {
		name string
		args args
		want []corev1.VolumeMount
	}{
		{
			name: "Test Heavy Forwarder Mounts",
			args: args{
				instance: &sfv1alpha1.SplunkForwarder{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: sfv1alpha1.SplunkForwarderSpec{
						UseHeavyForwarder: true,
					},
				},
			},
			want: []corev1.VolumeMount{
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunk/etc/apps/splunkauth/default",
				},
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunk/etc/apps/splunkauth/local",
				},
				{
					Name:      config.SplunkAuthSecretName,
					MountPath: "/opt/splunk/etc/apps/splunkauth/metadata",
				},
				{
					Name:      "test-hfconfig",
					MountPath: "/opt/splunk/etc/apps/osd_monitored_logs/local",
				},
				{
					Name:      "test-hfconfig",
					MountPath: "/opt/splunk/etc/apps/osd_monitored_logs/metadata",
				},
				{
					Name:      "splunk-state",
					MountPath: "/opt/splunk/var/lib",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHeavyForwarderVolumeMounts(tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHeavyForwarderVolumeMounts() = %v, want %v", got, tt.want)
			}
		})
	}
}
