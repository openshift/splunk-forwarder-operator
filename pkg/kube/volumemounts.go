package kube

import (
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeMounts returns []corev1.VolumeMount that tells where each secret, configmap, and host mount
// gets mounted in the container
func GetVolumeMounts(instance *sfv1alpha1.SplunkForwarder) []corev1.VolumeMount {
	var forwarderConfig string
	var mountPropagationMode = corev1.MountPropagationHostToContainer
	if instance.Spec.UseHeavyForwarder {
		forwarderConfig = instance.Name + "-internalsplunk"
	} else {
		forwarderConfig = config.SplunkAuthSecretName
	}

	return []corev1.VolumeMount{

		// Splunk Forwarder outputs and credentials
		{
			Name:      "splunk-auth-app",
			MountPath: "/opt/splunkforwarder/etc/apps/splunkauth",
		},
		{
			Name:      forwarderConfig,
			MountPath: "/opt/splunkforwarder/etc/apps/splunkauth/default",
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

		// Trust Bundle Mount
		{
			Name:      "trusted-ca-bundle",
			MountPath: "/etc/pki/ca-trust/source/anchors",
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

		// Inputs and Props Transform Mount
		{
			Name:      instance.Name + "-hfconfig",
			MountPath: "/opt/splunk/etc/apps/osd_monitored_logs/local",
		},
		{
			Name:      instance.Name + "-hfconfig",
			MountPath: "/opt/splunk/etc/apps/osd_monitored_logs/metadata",
		},

		// State Mount (https://www.splunk.com/en_us/blog/tips-and-tricks/what-is-this-fishbucket-thing.html)
		{
			Name:      "splunk-state",
			MountPath: "/opt/splunk/var/lib",
		},

		// Trust Bundle Mount
		{
			Name:      "trusted-ca-bundle",
			MountPath: "/etc/pki/ca-trust/source/anchors",
			ReadOnly:  true,
		},
	}
}
