package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes Returns an array of corev1.Volumes we want to attach
// It contains configmaps, secrets, and the host mount
func GetVolumes() []corev1.Volume {
	var hostPathDirectoryTypeForPtr = corev1.HostPathDirectory
	return []corev1.Volume{
		{
			Name: config.SplunkAuthSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: config.SplunkAuthSecretName,
				},
			},
		},

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
			Name: "host",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
					Type: &hostPathDirectoryTypeForPtr,
				},
			},
		},
	}
}
