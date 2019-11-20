package kube

import (
	"strconv"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GenerateConfigMaps generates config maps based on the values in our CRD
func GenerateConfigMaps(instance *sfv1alpha1.SplunkForwarder, namespacedName types.NamespacedName, clusterid string) []*corev1.ConfigMap {
	ret := []*corev1.ConfigMap{}

	metadataCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-metadata",
			Namespace: namespacedName.Namespace,
			Labels: map[string]string{
				"app": namespacedName.Name,
			},
			Annotations: map[string]string{
				"genVersion": strconv.FormatInt(instance.Generation, 10),
			},
		},
		Data: map[string]string{
			"local.meta": `
[]
access = read : [ * ], write : [ admin ]
export = system
`,
		},
	}
	ret = append(ret, metadataCM)

	inputsStr := ""

	for _, input := range instance.Spec.SplunkInputs {
		// No path passed in, skip it
		if input.Path == "" {
			continue
		}

		inputsStr += "[monitor://" + input.Path + "]\n"
		if input.SourceType != "" {
			inputsStr += "sourcetype = " + input.SourceType + "\n"
		} else {
			inputsStr += "sourcetype = _json\n"
		}

		if input.Index != "" {
			inputsStr += "index = " + input.Index + "\n"
		} else {
			inputsStr += "index = main\n"
		}

		if input.WhiteList != "" {
			inputsStr += "whitelist = " + input.WhiteList + "\n"
		}

		if input.BlackList != "" {
			inputsStr += "blacklist = " + input.BlackList + "\n"
		}

		if clusterid != "" {
			inputsStr += "_meta = clusterid::" + clusterid + "\n"
		}

		inputsStr += "disabled = false\n"
		inputsStr += "\n"
	}

	localCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-local",
			Namespace: namespacedName.Namespace,
			Labels: map[string]string{
				"app": namespacedName.Name,
			},
			Annotations: map[string]string{
				"genVersion": strconv.FormatInt(instance.Generation, 10),
			},
		},
		Data: map[string]string{
			"app.conf": `
[install]
state = enabled

[package]
check_for_updates = false

[ui]
is_visible = false
is_manageable = false
`,
			"inputs.conf": inputsStr,
		},
	}

	ret = append(ret, localCM)

	return ret
}
