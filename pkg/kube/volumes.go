package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes Returns an array of corev1.Volumes we want to attach
// It contains configmaps, secrets, and the host mount
func GetVolumes(mountHost bool, mountSecret bool, instanceName string) []corev1.Volume {
	volumes := []corev1.Volume{
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
	}

	if mountHost == true {
		var hostPathDirectoryTypeForPtr = corev1.HostPathDirectory
		volumes = append(volumes,
			corev1.Volume{
				Name: "host",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/",
						Type: &hostPathDirectoryTypeForPtr,
					},
				},
			})
	}

	if mountSecret == true {
		volumes = append(volumes,
			corev1.Volume{
				Name: config.SplunkAuthSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: config.SplunkAuthSecretName,
					},
				},
			})
	} else {
		// if we aren't mounting the secret, we're fwding to the splunk hf
		var internalName = instanceName + "-internalsplunk"
		volumes = append(volumes,
			corev1.Volume{
				Name: internalName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: internalName,
						},
					},
				},
			})
	}

	return volumes
}
