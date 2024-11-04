package kube

import (
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeMounts returns []corev1.VolumeMount that tells where each secret, configmap, and host mount
// gets mounted in the container
func GetVolumeMounts(instance *sfv1alpha1.SplunkForwarder, useHECToken bool) []corev1.VolumeMount {
	mountPropagationMode := corev1.MountPropagationHostToContainer

	volumeMounts := []corev1.VolumeMount{}
	if useHECToken {
		hecConfigMount := corev1.VolumeMount{
			Name:      config.SplunkHECTokenSecretName,
			MountPath: "/opt/splunkforwarder/etc/local",
		}
		volumeMounts = append(volumeMounts, hecConfigMount)
	} else {
		forwarderConfig := config.SplunkAuthSecretName
		if instance.Spec.UseHeavyForwarder {
			forwarderConfig = instance.Name + "-internalsplunk"
		}
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
	}
}
