package splunkforwarder

import (
	"context"
	"reflect"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
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

func testSplunkForwarderCR(useHeavyForwarder bool) *sfv1alpha1.SplunkForwarder {
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
			UseHeavyForwarder:     useHeavyForwarder,
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
				testSplunkForwarderCR(false),
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
				testSplunkForwarderCR(false),
				testSplunkForwarderService(),
				testSplunkForwarderSecret(),
			},
		},
		{
			name: "Heavy forwarders",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      instanceName,
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(true),
				testSplunkForwarderSecret(),
			},
		},
		{
			name: "Heavy forwarders - Old Service",
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
				testSplunkForwarderCR(true),
				testSplunkForwarderService(),
				testSplunkForwarderSecret(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := configv1.AddToScheme(scheme.Scheme); err != nil {
				t.Errorf("ReconcileSplunkForwarder.Reconcile() error = %v", err)
			}
			proxy := &configv1.Proxy{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}}
			fakeClient := fakekubeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(proxy).WithRuntimeObjects(tt.localObjects...).Build()
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
