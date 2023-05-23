// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"log"
	"strings"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

var (
	resrcs         *resources.Resources
	defaultTimeout = 300
	restConfig     *rest.Config
	deploymentName = "splunk-forwarder-operator"
	operatorName   = "splunk-forwarder-operator"
	serviceNames   = []string{"splunk-forwarder-operator-metrics",
		"splunk-forwarder-operator-catalog"}
	clusterRoles = []string{
		"splunk-forwarder-operator",
		"splunk-forwarder-operator-og-admin",
		"splunk-forwarder-operator-og-edit",
		"splunk-forwarder-operator-og-view",
	}
	rolePrefix          = "splunk-forwarder-operator"
	clusterRoleBindings = []string{
		"splunk-forwarder-operator-clusterrolebinding",
	}
	splunkforwarder_names = []string{
		"osde2e-dedicated-admin-splunkforwarder-x",
		"osde2e-splunkforwarder-test-2",
	}
	testsplunkforwarder                  = "osde2e-splunkforwarder-test-2"
	dedicatedadminsplunkforwarder        = "osde2e-dedicated-admin-splunkforwarder-x"
	operatorNamespace             string = "openshift-splunk-forwarder-operator"
	operatorLockFile              string = "splunk-forwarder-operator-lock"
)

// Blocking SplunkForwarder Signal
var _ = ginkgo.Describe("[Suite: operators] [OSD] Splunk Forwarder Operator", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		// setup the k8s client
		restConfig, err := config.GetConfig()
		Expect(err).Should(BeNil(), "failed to get kubeconfig")
		resrcs, err = resources.New(restConfig)
		Expect(err).Should(BeNil(), "resources.New error")
	})
	ginkgo.It("is installed", func(ctx context.Context) {

		ginkgo.By("checking the namespace exists")
		err := resrcs.Get(ctx, operatorNamespace, operatorNamespace, &corev1.Namespace{})
		Expect(err).Should(BeNil(), "namespace %s not found", operatorNamespace)

		// Check that the clusterRoleBindings exist
		ginkgo.By("checking the clusterrolebindings exist")
		var rolebindings rbacv1.RoleBindingList
		err = resrcs.List(ctx, &rolebindings)
		Expect(err).Should(BeNil(), "failed to list rolebindings")
		found := false
		for _, rolebinding := range rolebindings.Items {
			if strings.HasPrefix(rolebinding.Name, rolePrefix) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "unable to find clusterrolebindings with prefix %s", rolePrefix)

		// Check that the clusterRoles exist
		ginkgo.By("checking the clusterrole exists")
		var clusterRoles rbacv1.ClusterRoleList
		err = resrcs.List(ctx, &clusterRoles)
		Expect(err).Should(BeNil(), "failed to list clusterroles")
		found = false
		for _, clusterRole := range clusterRoles.Items {
			if strings.HasPrefix(clusterRole.Name, rolePrefix) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "unable to find cluster role with prefix %s", rolePrefix)

		ginkgo.By("checking the services exist")
		for _, serviceName := range serviceNames {
			err = resrcs.Get(ctx, serviceName, operatorNamespace, &corev1.Service{})
			Expect(err).Should(BeNil(), "service %s/%s not found", operatorNamespace, serviceName)
		}

		ginkgo.By("checking the deployment exists and is available")
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: operatorNamespace}}
		err = wait.For(conditions.New(resrcs).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue))
		Expect(err).Should(BeNil(), "deployment %s not available", deploymentName)

		//TODO:  post osde2e-common library add configmap check
		//checkConfigMapLockfile(h, operatorNamespace, operatorLockFile)

		//TODO:  post osde2e-common library add csv check
		//checkClusterServiceVersion(h, operatorNamespace, operatorName)

		// TODO: post osde2e-common library add upgrade check
		//checkUpgrade(helper.New(), "openshift-splunk-forwarder-operator",
		//	"openshift-splunk-forwarder-operator", "splunk-forwarder-operator",
		//	"splunk-forwarder-operator-catalog")
	})

	ginkgo.It("admin should be able to manage SplunkForwarders CR", func(ctx context.Context) {
		sf := makeMinimalSplunkforwarder("SplunkForwarder", "splunkforwarder.managed.openshift.io/v1alpha1", testsplunkforwarder)
		err := addSplunkforwarder(ctx, sf, "openshift-splunk-forwarder-operator")
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("test SplunkForwarders CR, if exists, should be deleted successfully", func(ctx context.Context) {
		err := deleteSplunkforwarder(ctx, testsplunkforwarder, operatorNamespace)
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("dedicated admin should not be able to manage SplunkForwarders CR", func(ctx context.Context) {

		sf := makeMinimalSplunkforwarder("SplunkForwarder", "splunkforwarder.managed.openshift.io/v1alpha1", dedicatedadminsplunkforwarder)
		err := dedicatedAaddSplunkforwarder(ctx, sf, "openshift-splunk-forwarder-operator")
		Expect(apierrors.IsForbidden(err)).To(BeTrue())
		if err == nil {
			err = deleteSplunkforwarder(ctx, dedicatedadminsplunkforwarder, operatorNamespace)
			if err != nil {
				log.Printf("Failed cleaning up %s CR in %s namespace", dedicatedadminsplunkforwarder, operatorNamespace)
			}
		}
	})

})

// Create test splunkforwarder CR definition
func makeMinimalSplunkforwarder(kind string, apiversion string, name string) sfv1alpha1.SplunkForwarder {
	sf := sfv1alpha1.SplunkForwarder{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiversion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
	return sf
}

// Create test splunkforwarder CR as a dedicated admin user
func dedicatedAaddSplunkforwarder(ctx context.Context, SplunkForwarder sfv1alpha1.SplunkForwarder, namespace string) error {
	restConfig, err := config.GetConfig()
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(SplunkForwarder.DeepCopy())
	if err != nil {
		return err
	}
	unstructuredObj := unstructured.Unstructured{obj}

	restConfig.Impersonate = rest.ImpersonationConfig{
		UserName: "test-user@redhat.com",
		Groups: []string{
			"dedicated-admins",
		},
	}
	defer func() {
		restConfig.Impersonate = rest.ImpersonationConfig{}
	}()
	_, err = Dynamic(restConfig).Resource(schema.GroupVersionResource{
		Group: "splunkforwarder.managed.openshift.io", Version: "v1alpha1", Resource: "splunkforwarders",
	}).Namespace(namespace).Create(ctx, &unstructuredObj, metav1.CreateOptions{})
	return (err)
}

// Create test splunkforwarder CR
func addSplunkforwarder(ctx context.Context, SplunkForwarder sfv1alpha1.SplunkForwarder, namespace string) error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(SplunkForwarder.DeepCopy())
	if err != nil {
		return err
	}
	unstructuredObj := unstructured.Unstructured{obj}
	restConfig, err := config.GetConfig()
	_, err = Dynamic(restConfig).Resource(schema.GroupVersionResource{
		Group: "splunkforwarder.managed.openshift.io", Version: "v1alpha1", Resource: "splunkforwarders",
	}).Namespace(namespace).Create(ctx, &unstructuredObj, metav1.CreateOptions{})
	return (err)
}

func deleteSplunkforwarder(ctx context.Context, name string, namespace string) error {
	restConfig, _ = config.GetConfig()
	_, err := Dynamic(restConfig).Resource(schema.GroupVersionResource{
		Group: "splunkforwarder.managed.openshift.io", Version: "v1alpha1", Resource: "splunkforwarders",
	}).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		e := Dynamic(restConfig).Resource(schema.GroupVersionResource{
			Group: "splunkforwarder.managed.openshift.io", Version: "v1alpha1", Resource: "splunkforwarders",
		}).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		log.Printf("Deleted splunkforwarder %s in namespace %s; Error:(%v)", name, operatorNamespace, e)
		return (e)
	}
	return nil
}

func Dynamic(restConfig *rest.Config) dynamic.Interface {
	client, err := dynamic.NewForConfig(restConfig)
	Expect(err).ShouldNot(HaveOccurred(), "failed to configure Dynamic client")
	return client
}
