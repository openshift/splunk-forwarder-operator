// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	"github.com/openshift/osde2e-common/pkg/gomega/assertions"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

var (
	resourceClient *resources.Resources

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
var _ = ginkgo.Describe("Splunk Forwarder Operator", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func() {
		// setup the k8s client
		restConfig, err := config.GetConfig()
		Expect(err).Should(BeNil(), "failed to get kubeconfig")
		resourceClient, err = resources.New(restConfig)
		Expect(err).Should(BeNil(), "resources.New error")
		Expect(sfv1alpha1.AddToScheme(resourceClient.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")

	})
	ginkgo.It("is installed", func(ctx context.Context) {

		ginkgo.By("checking the namespace exists")
		err := resourceClient.Get(ctx, operatorNamespace, operatorNamespace, &corev1.Namespace{})
		Expect(err).Should(BeNil(), "namespace %s not found", operatorNamespace)

		ginkgo.By("checking the clusterrolebindings exist")
		var rolebindings rbacv1.RoleBindingList
		err = resourceClient.List(ctx, &rolebindings)
		Expect(err).Should(BeNil(), "failed to list rolebindings")
		found := false
		for _, rolebinding := range rolebindings.Items {
			if strings.HasPrefix(rolebinding.Name, rolePrefix) {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "unable to find clusterrolebindings with prefix %s", rolePrefix)

		ginkgo.By("checking the clusterrole exists")
		var clusterRoles rbacv1.ClusterRoleList
		err = resourceClient.List(ctx, &clusterRoles)
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
			err = resourceClient.Get(ctx, serviceName, operatorNamespace, &corev1.Service{})
			Expect(err).Should(BeNil(), "service %s/%s not found", operatorNamespace, serviceName)
		}
		client, err := openshift.New()
		Expect(err).NotTo(HaveOccurred(), "Openshift client could not be created")

		ginkgo.By("checking the deployment exists and is available")
		assertions.EventuallyDeployment(ctx, client, deploymentName, operatorNamespace)

		ginkgo.By("checking the operator lock file config map exists")
		assertions.EventuallyConfigMap(ctx, client, operatorLockFile, operatorNamespace).WithTimeout(time.Duration(300)*time.Second).WithPolling(time.Duration(30)*time.Second).Should(Not(BeNil()), "configmap %s should exist", operatorLockFile)

		ginkgo.By("checking the operator CSV exists")
		restConfig, _ = config.GetConfig()
		clientset, err := olm.NewForConfig(restConfig)
		Expect(err).ShouldNot(HaveOccurred(), "failed to configure Operator clientset")
		EventuallyCsv(ctx, clientset, operatorName, operatorNamespace).WithTimeout(time.Duration(300)*time.Second).WithPolling(time.Duration(30)*time.Second).Should(BeTrue(), "CSV %s should exist", operatorName)

		// TODO: post osde2e-common library add upgrade check
		//checkUpgrade(helper.New(), "openshift-splunk-forwarder-operator",
		//	"openshift-splunk-forwarder-operator", "splunk-forwarder-operator",
		//	"splunk-forwarder-operator-catalog")
	})

	sf := makeMinimalSplunkforwarder(testsplunkforwarder)

	ginkgo.It("admin should be able to create SplunkForwarders CR", func(ctx context.Context) {
		err := resourceClient.WithNamespace(operatorNamespace).Create(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("admin should be able to delete SplunkForwarders CR", func(ctx context.Context) {
		err := resourceClient.Delete(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("dedicated admin should not be able to manage SplunkForwarders CR", func(ctx context.Context) {
		dsf := makeMinimalSplunkforwarder(dedicatedadminsplunkforwarder)
		restConfig, _ := config.GetConfig()
		u := User{Username: "test-user@redhat.com", Groups: []string{"dedicated-admins"}, RestConfig: restConfig}
		impersonatedResourceClient := u.NewImpersonatedClient()
		Expect(sfv1alpha1.AddToScheme(impersonatedResourceClient.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")
		err := impersonatedResourceClient.WithNamespace(operatorNamespace).Create(ctx, &dsf)
		Expect(apierrors.IsForbidden(err)).To(BeTrue())
	})

})

// Create test splunkforwarder CR definition
func makeMinimalSplunkforwarder(name string) sfv1alpha1.SplunkForwarder {
	sf := sfv1alpha1.SplunkForwarder{
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
	return sf
}
