package kube

import (
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateConfigMaps does stuff
func GenerateConfigMaps(inputs []sfv1alpha1.SplunkForwarderInputs, namespace string) []*corev1.ConfigMap {
	ret := []*corev1.ConfigMap{}

	metadataCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-metadata",
			Namespace: namespace,
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

	for input := 0; input < len(inputs); input++ {
		// No path passed in, skip it
		if inputs[input].Path == "" {
			continue
		}

		inputsStr += "[monitor://" + inputs[input].Path + "]\n"
		if inputs[input].SourceType != "" {
			inputsStr += "sourcetype = " + inputs[input].SourceType + "\n"
		} else {
			inputsStr += "sourcetype = _json\n"
		}

		if inputs[input].Index != "" {
			inputsStr += "index = " + inputs[input].Index + "\n"
		} else {
			inputsStr += "index = main\n"
		}

		if inputs[input].WhiteList != "" {
			inputsStr += "whitelist = " + inputs[input].WhiteList + "\n"
		}

		if inputs[input].BlackList != "" {
			inputsStr += "blacklist = " + inputs[input].BlackList + "\n"
		}

		inputsStr += "disabled = false\n"
		inputsStr += "\n"
	}

	localCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-local",
			Namespace: namespace,
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
