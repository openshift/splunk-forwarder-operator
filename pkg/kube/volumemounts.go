package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeMounts returns []corev1.VolumeMount that tells where each secret, configmap, and host mount
// gets mounted in the container
func GetVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		// Splunk Forwarder Certificate Mounts
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

		// Inputs Mount
		{
			Name:      "osd-monitored-logs-local",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
		},
		{
			Name:      "osd-monitored-logs-metadata",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
		},

		// Host Mount
		{
			Name:      "host",
			MountPath: "/host",
			ReadOnly:  true,
		},
	}
}
