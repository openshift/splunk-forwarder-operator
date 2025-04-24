// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	"github.com/openshift/osde2e-common/pkg/gomega/assertions"
	. "github.com/openshift/osde2e-common/pkg/gomega/matchers"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	k8s            *openshift.Client
	deploymentName = "splunk-forwarder-operator"
	operatorName   = "splunk-forwarder-operator"
	serviceNames   = []string{"splunk-forwarder-operator-metrics",
		"splunk-forwarder-operator-catalog"}
	rolePrefix                    = "splunk-forwarder-operator"
	testsplunkforwarder           = "osde2e-splunkforwarder-test-2"
	dedicatedadminsplunkforwarder = "osde2e-dedicated-admin-splunkforwarder-x"
	operatorNamespace             = "openshift-splunk-forwarder-operator"
	operatorLockFile              = "splunk-forwarder-operator-lock"
)

// Blocking SplunkForwarder Signal
var _ = ginkgo.Describe("Splunk Forwarder Operator", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		log.SetLogger(ginkgo.GinkgoLogr)
		var err error
		k8s, err = openshift.New(ginkgo.GinkgoLogr)
		Expect(err).ShouldNot(HaveOccurred(), "unable to setup k8s client")
		Expect(sfv1alpha1.AddToScheme(k8s.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")

	})
	ginkgo.It("is installed", func(ctx context.Context) {

		ginkgo.By("checking the namespace exists")
		err := k8s.Get(ctx, operatorNamespace, operatorNamespace, &corev1.Namespace{})
		Expect(err).Should(BeNil(), "namespace %s not found", operatorNamespace)

		ginkgo.By("checking the role exists")
		var roles rbacv1.RoleList
		err = k8s.WithNamespace(operatorNamespace).List(ctx, &roles)
		Expect(err).ShouldNot(HaveOccurred(), "failed to list roles")
		Expect(&roles).Should(ContainItemWithPrefix(rolePrefix), "unable to find roles with prefix %s", rolePrefix)

		ginkgo.By("checking the rolebinding exists")
		var rolebindings rbacv1.RoleBindingList
		err = k8s.List(ctx, &rolebindings)
		Expect(err).ShouldNot(HaveOccurred(), "failed to list rolebindings")
		Expect(&rolebindings).Should(ContainItemWithPrefix(rolePrefix), "unable to find rolebindings with prefix %s", rolePrefix)

		ginkgo.By("checking the clusterrole exists")
		var clusterRoles rbacv1.ClusterRoleList
		err = k8s.List(ctx, &clusterRoles)
		Expect(err).ShouldNot(HaveOccurred(), "failed to list clusterroles")
		Expect(&clusterRoles).Should(ContainItemWithPrefix(rolePrefix), "unable to find cluster role with prefix %s", rolePrefix)

		ginkgo.By("checking the clusterrolebinding exists")
		var clusterRoleBindings rbacv1.ClusterRoleBindingList
		err = k8s.List(ctx, &clusterRoleBindings)
		Expect(err).ShouldNot(HaveOccurred(), "unable to list clusterrolebindings")
		Expect(&clusterRoleBindings).Should(ContainItemWithPrefix(rolePrefix), "unable to find clusterrolebinding with prefix %s", rolePrefix)

		ginkgo.By("checking the services exist")
		for _, serviceName := range serviceNames {
			err = k8s.Get(ctx, serviceName, operatorNamespace, &corev1.Service{})
			Expect(err).Should(BeNil(), "unable to get service %s/%s", operatorNamespace, serviceName)
		}

		ginkgo.By("checking the deployment exists and is available")
		assertions.EventuallyDeployment(ctx, k8s, deploymentName, operatorNamespace)

		ginkgo.By("checking the operator lock file config map exists")
		assertions.EventuallyConfigMap(ctx, k8s, operatorLockFile, operatorNamespace).WithTimeout(time.Duration(300)*time.Second).WithPolling(time.Duration(30)*time.Second).Should(Not(BeNil()), "configmap %s should exist", operatorLockFile)

		ginkgo.By("checking the operator CSV has Succeeded")
		restConfig, _ := config.GetConfig()
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		Expect(err).ShouldNot(HaveOccurred(), "failed to configure Dynamic client")
		Eventually(func() bool {
			csvList, err := dynamicClient.Resource(
				schema.GroupVersionResource{
					Group:    "operators.coreos.com",
					Version:  "v1alpha1",
					Resource: "clusterserviceversions",
				},
			).Namespace(operatorNamespace).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to retrieve CSV from namespace %s", operatorNamespace)
			for _, csv := range csvList.Items {
				specName, _, _ := unstructured.NestedFieldCopy(csv.Object, "spec", "displayName")
				statusPhase, _, _ := unstructured.NestedFieldCopy(csv.Object, "status", "phase")
				if statusPhase == "Succeeded" && specName == operatorName {
					return true
				}
			}
			return false
		}).WithTimeout(time.Duration(300)*time.Second).WithPolling(time.Duration(30)*time.Second).Should(BeTrue(), "CSV %s should exist and have Succeeded status", operatorName)

	})

	ginkgo.It("admin should be able to create and delete SplunkForwarders CR", func(ctx context.Context) {
		sf := makeMinimalSplunkforwarder(testsplunkforwarder)
		err := k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
		err = k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("dedicated admin should not be able to manage SplunkForwarders CR", func(ctx context.Context) {
		dsf := makeMinimalSplunkforwarder(dedicatedadminsplunkforwarder)
		impersonatedResourceClient, _ := k8s.Impersonate("test-user@redhat.com", "dedicated-admins")
		Expect(sfv1alpha1.AddToScheme(impersonatedResourceClient.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")
		err := impersonatedResourceClient.WithNamespace(operatorNamespace).Create(ctx, &dsf)
		Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected err to be forbidden, got: %v", err)
	})

	ginkgo.PIt("can be upgraded", func(ctx context.Context) {
		ginkgo.By("forcing operator upgrade")
		err := k8s.UpgradeOperator(ctx, "openshift-"+operatorName, operatorNamespace)
		Expect(err).NotTo(HaveOccurred(), "operator upgrade failed")
	})

})

// Create test splunkforwarder CR definition
func makeMinimalSplunkforwarder(name string) sfv1alpha1.SplunkForwarder {
	return sfv1alpha1.SplunkForwarder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: operatorNamespace,
		},
		Spec: sfv1alpha1.SplunkForwarderSpec{
			SplunkLicenseAccepted: true,
			UseHeavyForwarder:     false,
			SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
				{
					Path: "",
				},
			},
		},
	}
}
