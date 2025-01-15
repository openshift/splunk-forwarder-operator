package kube

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
)

const (
	instanceName      = "test"
	instanceNamespace = "openshift-test"
	image             = "test-image"
	imageTag          = "0.0.1"
	imageDigest       = "sha256:2452a3f01e840661ee1194777ed5a9185ceaaa9ec7329ed364fa2f02be22a701"
)

// splunkForwarderInstance returns (a pointer to) a SplunkForwarder CR as input to
// GenerateDaemonSet. Parameters:
// - useTag: If true, the ImageTag field is set.
// - useDigest: If true, the ImageDigest field is set.
func splunkForwarderInstance(useDigest bool) *sfv1alpha1.SplunkForwarder {
	spec := sfv1alpha1.SplunkForwarderSpec{
		SplunkLicenseAccepted:  true,
		Image:                  image,
	}
	if useDigest {
		spec.ImageDigest = imageDigest
	} else {
		spec.ImageTag = imageTag
	}
	return &sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:       instanceName,
			Namespace:  instanceNamespace,
			Generation: 10,
		},
		Spec: spec,
	}
}

// DoDiff deep-compares two `runtime.Object`s and fails the `t`est with a useful message (showing
// the diff) if they differ.
func DeepEqualWithDiff(t *testing.T, expected, actual runtime.Object) {
	t.Helper()
	diff := cmp.Diff(expected, actual)
	if diff != "" {
		t.Fatal("Objects differ: -expected, +actual\n", diff)
	}
}
