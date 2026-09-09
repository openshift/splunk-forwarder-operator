package splunkforwarder

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	"github.com/openshift/splunk-forwarder-operator/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakekubeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	instanceName      = "test"
	instanceNamespace = "openshift-test"
	image             = "test-image"
	imageTag          = "0.0.1"
)

// TODO: tests should also check the reconciliation side-effects
// ie. making sure objects get created or modified properly
func testSplunkForwarderCR() *sfv1alpha1.SplunkForwarder {
	ret := &sfv1alpha1.SplunkForwarder{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SplunkForwarder",
			APIVersion: "splunkforwarder.managed.openshift.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: instanceNamespace,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkLicenseAccepted: true,
			Image:                 image,
			ImageTag:              imageTag,
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{
					Path: "/var/log/test",
				},
			},
		},
	}
	return ret
}

func testSplunkForwarderSecret() *corev1.Secret {
	ret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkAuthSecretName,
			Namespace: instanceNamespace,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
	}
	return ret
}

func testSplunkHECSecret() *corev1.Secret {
	ret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SplunkHECTokenSecretName,
			Namespace: instanceNamespace,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
	}
	return ret
}

func testSplunkForwarderService() *corev1.Service {
	ret := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: instanceNamespace,
			CreationTimestamp: metav1.Time{
				Time: time.Date(2019, 12, 01, 12, 12, 0, 0, time.UTC),
			},
		},
	}
	return ret
}

func TestReconcileSplunkForwarder_Reconcile(t *testing.T) {
	if err := sfv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("ReconcileSplunkForwarder.Reconcile() error = %v", err)
		return
	}
	if err := configv1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("ReconcileSplunkForwarder.Reconcile() error = %v", err)
		return
	}
	type args struct {
		request reconcile.Request
	}
	tests := []struct {
		name         string
		args         args
		want         reconcile.Result
		wantErr      bool
		localObjects []runtime.Object
	}{
		{
			name: "No CR",
			args: args{
				request: reconcile.Request{},
			},
			want:         reconcile.Result{},
			wantErr:      false,
			localObjects: []runtime.Object{},
		},
		{
			name: "No Secret",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      instanceName,
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: true,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
			},
		},
		{
			name: "No heavy forwarders",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      instanceName,
						Namespace: instanceNamespace,
					},
				},
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
				testSplunkForwarderService(),
				testSplunkForwarderSecret(),
			},
		},
		{
			name: "HEC secret present",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      instanceName,
						Namespace: instanceNamespace,
					},
				},
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
				testSplunkForwarderService(),
				testSplunkForwarderSecret(),
				testSplunkHECSecret(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakekubeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(tt.localObjects...).Build()
			r := &SplunkForwarderReconciler{
				Client:    fakeClient,
				Scheme:    scheme.Scheme,
				ReqLogger: log.WithValues(),
			}
			got, err := r.Reconcile(context.TODO(), tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileSplunkForwarder.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileSplunkForwarder.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileUpdatesStaleConfigMap(t *testing.T) {
	if err := sfv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}
	if err := configv1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	cr := &sfv1alpha1.SplunkForwarder{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SplunkForwarder",
			APIVersion: "splunkforwarder.managed.openshift.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 37,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkLicenseAccepted: true,
			Image:                 image,
			ImageTag:              imageTag,
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{Path: "/var/log/audit.log", Index: "audit"},
			},
			Filters: []sfv1alpha1.SplunkFilter{
				{Name: "ignore_sa", Filter: `"user":{"username":"system:serviceaccount:[^"]+"}`},
			},
		},
	}

	staleCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-local",
			Namespace: instanceNamespace,
			Annotations: map[string]string{
				"genVersion": "37",
			},
			Labels: map[string]string{"app": instanceName},
		},
		Data: map[string]string{
			"app.conf":    "old",
			"inputs.conf": "old",
			"props.conf":  fmt.Sprintf("\n[_json]\nTRUNCATE = %d\n", kube.MaxEventSize),
		},
	}

	metaCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osd-monitored-logs-metadata",
			Namespace: instanceNamespace,
			Annotations: map[string]string{
				"genVersion": "37",
			},
			Labels: map[string]string{"app": instanceName},
		},
		Data: map[string]string{
			"local.meta": "\n[]\naccess = read : [ * ], write : [ admin ]\nexport = system\n",
		},
	}

	fakeClient := fakekubeclient.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithRuntimeObjects(cr, testSplunkForwarderSecret(), staleCM, metaCM).
		Build()

	r := &SplunkForwarderReconciler{
		Client:    fakeClient,
		Scheme:    scheme.Scheme,
		ReqLogger: log.WithValues(),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: instanceName, Namespace: instanceNamespace},
	}

	result, err := r.Reconcile(context.TODO(), req)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result == (reconcile.Result{}) {
		t.Error("expected non-empty reconcile result after updating stale configmap")
	}

	updated := &corev1.ConfigMap{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "osd-monitored-logs-local", Namespace: instanceNamespace}, updated)
	if err != nil {
		t.Fatalf("failed to get updated configmap: %v", err)
	}
	if _, ok := updated.Data["transforms.conf"]; !ok {
		t.Error("expected transforms.conf in updated configmap")
	}
	if !strings.Contains(updated.Data["props.conf"], "TRANSFORMS-null") {
		t.Error("expected TRANSFORMS-null in updated props.conf")
	}
}
