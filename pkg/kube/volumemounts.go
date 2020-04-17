package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeMounts returns []corev1.VolumeMount that tells where each secret, configmap, and host mount
// gets mounted in the container
func GetVolumeMounts(instance *sfv1alpha1.SplunkForwarder, mountSecret bool) []corev1.VolumeMount {
	var forwarderConfig string
	if instance.Spec.UseHeavyForwarder == true && !mountSecret {
		forwarderConfig = instance.Name + "-internalsplunk"
	} else {
		forwarderConfig = config.SplunkAuthSecretName
	}
	return []corev1.VolumeMount{
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

// GetHeavyForwarderVolumeMounts returns []corev1.VolumeMount that tells where each secret, and configmap gets mounted in the container
func GetHeavyForwarderVolumeMounts(instance *sfv1alpha1.SplunkForwarder) []corev1.VolumeMount {
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

		// Inputs and Props Transform Mount
		{
			Name:      instance.Name + "-hfconfig",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/local",
		},
		{
			Name:      instance.Name + "-hfconfig",
			MountPath: "/opt/splunkforwarder/etc/apps/osd_monitored_logs/metadata",
		},
	}
}
