package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes Returns an array of corev1.Volumes we want to attach
// It contains configmaps, secrets, and the host mount
func GetVolumes(mountHost, mountSecret, mountHECToken bool, instanceName string) []corev1.Volume {
	var hostPathDirectoryTypeForPtr = corev1.HostPathDirectory

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
		{
			Name: "splunk-state",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/misc",
					Type: &hostPathDirectoryTypeForPtr,
				},
			},
		},
	}

	if mountHost {
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
	} else {
		// if we aren't mounting the host dir, we're the hf
		var hfName = instanceName + "-hfconfig"
		volumes = append(volumes,
			corev1.Volume{
				Name: hfName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hfName,
						},
					},
				},
			})
	}

	if mountHECToken {
		hecVolumes := []corev1.Volume{
			{
				Name: config.SplunkHECTokenSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: config.SplunkHECTokenSecretName,
					},
				},
			},
			{
				Name: "splunk-config",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		volumes = append(volumes, hecVolumes...)
	} else if mountSecret {
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
