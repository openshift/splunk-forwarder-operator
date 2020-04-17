package kube

import (
	"strconv"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateService returns a service for the Heavy Forwarder so that the UF can connect to it
func GenerateService(instance *sfv1alpha1.SplunkForwarder) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name + "-hf",
			},
			Annotations: map[string]string{
				"genVersion": strconv.FormatInt(instance.Generation, 10),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Selector: map[string]string{
				"app": "splunk-hf",
			},
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     9997,
				},
			},
		},
	}
}
