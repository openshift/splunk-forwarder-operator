package kube

import (
	"github.com/openshift/splunk-forwarder-operator/config"
	corev1 "k8s.io/api/core/v1"
)

func buildProjectedSecretVolume(useHEC bool) corev1.Volume {

	s := corev1.SecretProjection{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: config.SplunkAuthSecretName,
		},
		Items: []corev1.KeyToPath{{
			Key:  "cacert.pem",
			Path: "cacert.pem",
		},
		},
	}

	if useHEC {
		s.Items = append(s.Items, corev1.KeyToPath{
			Key:  "outputs-hec.conf",
			Path: "outputs.conf",
		})
	} else {
		s.Items = append(s.Items, []corev1.KeyToPath{
			{
				Key:  "server.pem",
				Path: "server.pem",
			},
			{
				Key:  "outputs.conf",
				Path: "outputs.conf",
			},
		}...)
	}

	return corev1.Volume{
		Name: config.SplunkAuthSecretName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{{Secret: &s}},
			},
		},
	}

}

// GetVolumes Returns an array of corev1.Volumes we want to attach
// It contains configmaps, secrets, and the host mount
func GetVolumes(mountHost bool, mountSecret bool, useHEC bool, instanceName string) []corev1.Volume {
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
		{
			Name: "trusted-ca-bundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "trusted-ca-bundle",
					},
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

	if mountSecret {
		// we use an emptydir for the app, so it can write a local encrypted token if necessary
		volumes = append(volumes, corev1.Volume{
			Name: "splunk-auth-app",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumDefault,
				},
			},
		})
		// build a projected secret volume from the splunkauth secret
		volumes = append(volumes, buildProjectedSecretVolume(useHEC))
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
