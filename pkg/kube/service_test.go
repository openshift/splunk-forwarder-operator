package kube

import (
	"reflect"
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateService(t *testing.T) {
	type args struct {
		instance *sfv1alpha1.SplunkForwarder
	}
	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "Testing Service Generation",
			args: args{
				instance: &sfv1alpha1.SplunkForwarder{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "testing",
						Namespace:  "openshift-test",
						Generation: 1,
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testing",
					Namespace: "openshift-test",
					Labels: map[string]string{
						"name": "splunk-heavy-forwarder-service",
					},
					Annotations: map[string]string{
						"genVersion": "1",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: "ClusterIP",
					Selector: map[string]string{
						"name": "splunk-heavy-forwarder",
					},
					Ports: []corev1.ServicePort{
						{
							Protocol: "TCP",
							Port:     9997,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateService(tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateService() = %v, want %v", got, tt.want)
			}
		})
	}
}
