package kube

import (
	configv1 "github.com/openshift/api/config/v1"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeMounts returns []corev1.VolumeMount that tells where each secret, configmap, and host mount
// gets mounted in the container
func GetVolumeMounts(instance *sfv1alpha1.SplunkForwarder, useHECToken bool, proxyConfig *configv1.Proxy) []corev1.VolumeMount {
	mountPropagationMode := corev1.MountPropagationHostToContainer

	volumeMounts := []corev1.VolumeMount{}
	if useHECToken {
		hecConfigMount := corev1.VolumeMount{
			Name:      "splunk-config",
			MountPath: "/opt/splunkforwarder/etc/system/local",
		}
		volumeMounts = append(volumeMounts, hecConfigMount)
	} else {
		forwarderConfig := config.SplunkAuthSecretName
		splunkConfigMounts := []corev1.VolumeMount{
			// Splunk Forwarder Certificate Mounts
			{
				Name:      forwarderConfig,
				MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/default",
			},
			{
				Name:      forwarderConfig,
				MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/local",
			},
			{
				Name:      forwarderConfig,
				MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/metadata",
			},
		}
		volumeMounts = append(volumeMounts, splunkConfigMounts...)
	}

	if proxyConfig != nil && (proxyConfig.Spec.HTTPProxy != "" || proxyConfig.Spec.HTTPSProxy != "") {
		proxyConfigMount := corev1.VolumeMount{
			Name:      instance.Name + "-proxy",
			MountPath: "/opt/splunkforwarder/etc/system/local", // !!MAY BREAK!!
		}
		volumeMounts = append(volumeMounts, proxyConfigMount)
	}

	defaultMounts := []corev1.VolumeMount{
		// Inputs Mount
		{
			Name:      "osd-monitored-logs-local",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
		},
		{
			Name:      "osd-monitored-logs-metadata",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
		},

		// State Mount
		{
			Name:      "splunk-state",
			MountPath: "/opt/splunkforwarder/var/lib",
		},

		// Host Mount
		{
			Name:             "host",
			MountPath:        "/host",
			MountPropagation: &mountPropagationMode,
			ReadOnly:         true,
		},
	}
	volumeMounts = append(volumeMounts, defaultMounts...)
	return volumeMounts
}

func getInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      config.SplunkHECTokenSecretName,
			MountPath: "/tmp/splunk-hec-token",
		},
		{
			Name:      "splunk-config",
			MountPath: "/tmp/splunk-config",
		},
	}
}
