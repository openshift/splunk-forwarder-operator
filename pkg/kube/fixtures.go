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
	hfImage           = "test-hf-image"
	imageTag          = "0.0.1"
	imageDigest       = "sha256:2452a3f01e840661ee1194777ed5a9185ceaaa9ec7329ed364fa2f02be22a701"
	heavyDigest       = "sha256:49b40c2c5d79913efb7eff9f3bf9c7348e322f619df10173e551b2596913d52a"
)

// splunkForwarderInstance returns (a pointer to) a SplunkForwarder CR as input to
// GenerateDaemonSet. Parameters:
// - useHeavy: The UseHeavyForwarder field is set to this value.
// - useTag: If true, the ImageTag field is set.
// - useDigest: If true, the ImageDigest field is set.
// - useHFDigest: If true, the HeavyForwarderDigest field is set.
func splunkForwarderInstance(useHeavy, useTag, useDigest, useHFDigest bool) *sfv1alpha1.SplunkForwarder {
	spec := sfv1alpha1.SplunkForwarderSpec{
		UseHeavyForwarder:      useHeavy,
		HeavyForwarderReplicas: 0,
		SplunkLicenseAccepted:  true,
		HeavyForwarderSelector: "infra",
		Image:                  image,
		HeavyForwarderImage:    hfImage,
	}
	if useTag {
		spec.ImageTag = imageTag
	}
	if useDigest {
		spec.ImageDigest = imageDigest
	}
	if useHFDigest {
		spec.HeavyForwarderDigest = heavyDigest
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
