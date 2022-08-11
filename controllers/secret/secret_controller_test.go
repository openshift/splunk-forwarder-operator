package secret

import (
	"context"
	"reflect"
	"testing"
	"time"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	"github.com/openshift/splunk-forwarder-operator/config"
	appsv1 "k8s.io/api/apps/v1"
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

func testSplunkForwarderDS() *appsv1.DaemonSet {
	ret := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName + "-ds",
			Namespace: instanceNamespace,
			CreationTimestamp: metav1.Time{
				Time: time.Date(2019, 12, 01, 12, 12, 0, 0, time.UTC),
			},
		},
	}
	return ret
}

func TestReconcileSecret_Reconcile(t *testing.T) {
	if err := sfv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("SecretReconciler.Reconcile() error = %v", err)
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
			name: "No CRD",
			args: args{
				request: reconcile.Request{},
			},
			want:         reconcile.Result{},
			wantErr:      false,
			localObjects: []runtime.Object{},
		},
		{
			name: "Invalid Secret Name",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "invalid-secret-name",
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
			},
		},
		{
			name: "No secret",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      config.SplunkAuthSecretName,
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
			},
		},
		{
			name: "No daemonset",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      config.SplunkAuthSecretName,
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
				testSplunkForwarderSecret(),
			},
		},
		{
			name: "Daemonset Timestamp",
			args: args{
				request: reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      config.SplunkAuthSecretName,
						Namespace: instanceNamespace,
					},
				},
			},
			want:    reconcile.Result{},
			wantErr: false,
			localObjects: []runtime.Object{
				testSplunkForwarderCR(),
				testSplunkForwarderSecret(),
				testSplunkForwarderDS(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakekubeclient.NewFakeClientWithScheme(scheme.Scheme, tt.localObjects...)
			r := &SecretReconciler{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			got, err := r.Reconcile(context.TODO(), tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SecretReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
