package kube

import (
	"reflect"
	"testing"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/pkg/apis/splunkforwarder/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGenerateConfigMaps(t *testing.T) {
	var testInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{
					Path:      "",
					Index:     "test-index",
					WhiteList: ".*log$",
					BlackList: ".*bak$",
				},
				{
					Path:      "/var/derp",
					Index:     "test-index",
					WhiteList: ".*log$",
					BlackList: ".*bak$",
				},
				{
					Path:       "/var/derp.text",
					SourceType: "text",
					WhiteList:  ".*log$",
					BlackList:  ".*bak$",
				},
			},
		},
	}
	type args struct {
		instance       *sfv1alpha1.SplunkForwarder
		namespacedName types.NamespacedName
		clusterid      string
	}
	tests := []struct {
		name string
		args args
		want []*corev1.ConfigMap
	}{
		{
			name: "Test Generate Config Maps",
			args: args{
				instance:       testInstance,
				namespacedName: types.NamespacedName{Namespace: instanceNamespace, Name: instanceName},
				clusterid:      "test",
			},
			want: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osd-monitored-logs-metadata",
						Namespace: instanceNamespace,
						Labels: map[string]string{
							"app": instanceName,
						},
						Annotations: map[string]string{
							"genVersion": "10",
						},
					},
					Data: map[string]string{
						"local.meta": `
[]
access = read : [ * ], write : [ admin ]
export = system
`,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osd-monitored-logs-local",
						Namespace: testInstance.Namespace,
						Labels: map[string]string{
							"app": testInstance.Name,
						},
						Annotations: map[string]string{
							"genVersion": "10",
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
						"inputs.conf": `[monitor:///var/derp]
sourcetype = _json
index = test-index
whitelist = .*log$
blacklist = .*bak$
_meta = clusterid::test
disabled = false

[monitor:///var/derp.text]
sourcetype = text
index = main
whitelist = .*log$
blacklist = .*bak$
_meta = clusterid::test
disabled = false

`,
					"props.conf": `
[_json]
TRUNCATE = 1000000
`,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateConfigMaps(tt.args.instance, tt.args.namespacedName, tt.args.clusterid); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateConfigMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateInternalConfigMap(t *testing.T) {
	var testInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{},
	}
	type args struct {
		instance       *sfv1alpha1.SplunkForwarder
		namespacedName types.NamespacedName
	}
	tests := []struct {
		name string
		args args
		want *corev1.ConfigMap
	}{
		{
			name: "Test internal config map",
			args: args{
				instance:       testInstance,
				namespacedName: types.NamespacedName{Namespace: instanceNamespace, Name: instanceName},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName + "-internalsplunk",
					Namespace: instanceNamespace,
					Labels: map[string]string{
						"app": instanceName,
					},
					Annotations: map[string]string{
						"genVersion": "10",
					},
				},
				Data: map[string]string{
					"local.meta": `
[]
access = read : [ * ], write : [ admin ]
export = system
`,
					"outputs.conf": `
[tcpout]
defaultGroup = internal

[tcpout:internal]
server = test:9997
`,
					"limits.conf": `
[thruput]
maxKBps = 0
`,
					"props.conf": `
[_json]
TRUNCATE = 1000000
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateInternalConfigMap(tt.args.instance, tt.args.namespacedName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateInternalConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateFilteringConfigMap(t *testing.T) {
	var testInstance = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{},
	}
	var testInstanceFilters = &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			Filters: []sfv1alpha1.SplunkFilter{
				{
					Name:   "ignore_chatty_system_users",
					Filter: `"user":{"username":"system:(?:kube-(?:controller-manager|scheduler|apiserver-cert-syncer)|apiserver|aggregator)"`,
				},
			},
		},
	}
	type args struct {
		instance       *sfv1alpha1.SplunkForwarder
		namespacedName types.NamespacedName
	}
	tests := []struct {
		name string
		args args
		want *corev1.ConfigMap
	}{
		{
			name: "No filters",
			args: args{
				instance:       testInstance,
				namespacedName: types.NamespacedName{Namespace: instanceNamespace, Name: instanceName},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName + "-hfconfig",
					Namespace: instanceNamespace,
					Labels: map[string]string{
						"app": instanceName,
					},
					Annotations: map[string]string{
						"genVersion": "10",
					},
				},
				Data: map[string]string{
					"local.meta": `
[]
access = read : [ * ], write : [ admin ]
export = system
`,
					"inputs.conf": `
[splunktcp]
route = has_key:_replicationBucketUUID:replicationQueue;has_key:_dstrx:typingQueue;has_key:_linebreaker:typingQueue;absent_key:_linebreaker:parsingQueue

[splunktcp://:9997]
connection_host = dns
`,
					"limits.conf": `
[thruput]
maxKBps = 0
`,
					"props.conf": `
[_json]
TRUNCATE = 1000000
`,
				},
			},
		},
		{
			name: "Filters",
			args: args{
				instance:       testInstanceFilters,
				namespacedName: types.NamespacedName{Namespace: testInstance.Namespace, Name: testInstance.Name},
			},
			want: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName + "-hfconfig",
					Namespace: instanceNamespace,
					Labels: map[string]string{
						"app": instanceName,
					},
					Annotations: map[string]string{
						"genVersion": "10",
					},
				},
				Data: map[string]string{
					"local.meta": `
[]
access = read : [ * ], write : [ admin ]
export = system
`,
					"inputs.conf": `
[splunktcp]
route = has_key:_replicationBucketUUID:replicationQueue;has_key:_dstrx:typingQueue;has_key:_linebreaker:typingQueue;absent_key:_linebreaker:parsingQueue

[splunktcp://:9997]
connection_host = dns
`,
					"limits.conf": `
[thruput]
maxKBps = 0
`,
					"props.conf": `
[_json]
TRUNCATE = 1000000
TRANSFORMS-null =filter_ignore_chatty_system_users `,
					"transforms.conf": `[filter_ignore_chatty_system_users]
DEST_KEY = queue
FORMAT = nullQueue
REGEX = "user":{"username":"system:(?:kube-(?:controller-manager|scheduler|apiserver-cert-syncer)|apiserver|aggregator)"

`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateFilteringConfigMap(tt.args.instance, tt.args.namespacedName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateFilteringConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
