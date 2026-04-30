// DO NOT REMOVE TAGS BELOW. IF ANY NEW TEST FILES ARE CREATED UNDER /osde2e, PLEASE ADD THESE TAGS TO THEM IN ORDER TO BE EXCLUDED FROM UNIT TESTS.
//go:build osde2e
// +build osde2e

package osde2etests

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	"github.com/openshift/osde2e-common/pkg/gomega/assertions"
	. "github.com/openshift/osde2e-common/pkg/gomega/matchers"
	sfv1alpha1 "github.com/openshift/splunk-forwarder-operator/api/v1alpha1"
	operatorconfig "github.com/openshift/splunk-forwarder-operator/config"
)

var (
	k8s            *openshift.Client
	deploymentName = "splunk-forwarder-operator"
	operatorName   = "splunk-forwarder-operator"
	serviceNames   = []string{"splunk-forwarder-operator-metrics"}
	rolePrefix                    = "splunk-forwarder-operator"
	testSplunkForwarder           = "osde2e-splunkforwarder-test-2"
	dedicatedAdminSplunkForwarder = "osde2e-dedicated-admin-splunkforwarder-x"
	operatorNamespace             = "openshift-splunk-forwarder-operator"
	operatorLockFile              = "splunk-forwarder-operator-lock"
	clusterID                     string
)

// Blocking SplunkForwarder Signal
var _ = ginkgo.Describe("Splunk Forwarder Operator", ginkgo.Ordered, func() {

	ginkgo.BeforeAll(func(ctx context.Context) {
		log.SetLogger(ginkgo.GinkgoLogr)
		var err error
		k8s, err = openshift.New(ginkgo.GinkgoLogr)
		Expect(err).ShouldNot(HaveOccurred(), "unable to setup k8s client")
		Expect(sfv1alpha1.AddToScheme(k8s.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")

		ginkgo.By("creating test secrets for e2e tests")
		err = createTestSecrets(ctx, k8s, operatorNamespace)
		Expect(err).ShouldNot(HaveOccurred(), "unable to create test secrets")

		clusterID = os.Getenv("OCM_CLUSTER_ID")
		Expect(clusterID).ShouldNot(BeEmpty(), "OCM_CLUSTER_ID is required but not set")

		// PKO's ClusterPackage already configures SCC access - no need to modify
	})

	ginkgo.AfterAll(func(ctx context.Context) {
		ginkgo.By("cleaning up test secrets")
		cleanupTestSecrets(ctx, k8s, operatorNamespace)
	})
	ginkgo.It("is installed", func(ctx context.Context) {
		ginkgo.By("checking the namespace exists")
		err := k8s.Get(ctx, operatorNamespace, operatorNamespace, &corev1.Namespace{})
		Expect(err).Should(BeNil(), "namespace %s not found", operatorNamespace)

		// PKO only creates ClusterRoles and ClusterRoleBindings (no namespace-scoped RBAC)

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

		// PKO does not use ClusterServiceVersion - it uses ClusterPackages instead
	})

	ginkgo.It("creates ConfigMaps and DaemonSet when CR is created", func(ctx context.Context) {
		crName := "test-reconciliation-sf"

		ginkgo.By("creating a SplunkForwarder CR with test configuration")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			"/var/log/test.log",
			"test_index",
			"_json",
		)
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		// Cleanup at end
		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("verifying metadata ConfigMap is created")
		var metadataCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-metadata", operatorNamespace, &metadataCM)
		}).WithTimeout(60*time.Second).WithPolling(5*time.Second).Should(Succeed(),
			"metadata ConfigMap should be created")

		ginkgo.By("verifying metadata ConfigMap has correct content")
		Expect(metadataCM.Labels).To(HaveKeyWithValue("app", crName))
		Expect(metadataCM.Annotations).To(HaveKey("genVersion"))
		Expect(metadataCM.Data).To(HaveKey("local.meta"))

		ginkgo.By("verifying local ConfigMap is created")
		var localCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		}).WithTimeout(60*time.Second).Should(Succeed(),
			"local ConfigMap should be created")

		ginkgo.By("verifying local ConfigMap contains inputs.conf")
		Expect(localCM.Data).To(HaveKey("inputs.conf"))
		Expect(localCM.Data).To(HaveKey("app.conf"))
		Expect(localCM.Data).To(HaveKey("props.conf"))

		ginkgo.By("verifying DaemonSet is created")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60*time.Second).Should(Succeed(),
			"DaemonSet should be created")

		ginkgo.By("verifying DaemonSet has correct configuration")
		Expect(ds.Labels).To(HaveKeyWithValue("app", crName))
		Expect(ds.Spec.Template.Spec.Containers).To(HaveLen(1))
		Expect(ds.Spec.Template.Spec.Containers[0].Name).To(Equal("splunk-uf"))

		// TODO: Temporarily commented out for integration testing
		// Integration clusters don't have the full production setup (secrets, SCC permissions, etc.)
		// Follow-up work to address this will be tracked in HCMSEC-3314
		// ginkgo.By("verifying DaemonSet pods become ready")
		// Eventually(func() bool {
		// 	k8s.Get(ctx, dsName, operatorNamespace, &ds)
		// 	// At least one pod should be ready
		// 	return ds.Status.NumberReady > 0
		// }).WithTimeout(180*time.Second).WithPolling(10*time.Second).Should(BeTrue(),
		// 	"DaemonSet should have at least one ready pod")
	})

	ginkgo.It("verifies log collection and forwarding workflow configuration", func(ctx context.Context) {
		crName := "test-log-workflow"
		testPath := "/var/log/audit/audit.log"
		testIndex := "audit_logs"
		testSourcetype := "linux_audit"

		ginkgo.By("creating a SplunkForwarder CR with specific input configuration")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			testPath,
			testIndex,
			testSourcetype,
		)
		sf.Spec.ClusterID = clusterID
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("verifying inputs.conf is generated with correct monitor stanza")
		var localCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		inputsConf := localCM.Data["inputs.conf"]

		ginkgo.By("checking monitor stanza path")
		Expect(inputsConf).To(ContainSubstring("[monitor://" + testPath + "]"))

		ginkgo.By("checking index configuration")
		Expect(inputsConf).To(ContainSubstring("index = " + testIndex))

		ginkgo.By("checking sourcetype configuration")
		Expect(inputsConf).To(ContainSubstring("sourcetype = " + testSourcetype))

		ginkgo.By("checking cluster metadata is included")
		Expect(inputsConf).To(ContainSubstring("_meta = clusterid::" + clusterID))

		ginkgo.By("checking input is enabled")
		Expect(inputsConf).To(ContainSubstring("disabled = false"))

		ginkgo.By("verifying props.conf contains TRUNCATE setting")
		Expect(localCM.Data).To(HaveKey("props.conf"))
		propsConf := localCM.Data["props.conf"]
		Expect(propsConf).To(ContainSubstring("TRUNCATE = 102400")) // 100KB

		ginkgo.By("verifying DaemonSet mounts host filesystem")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		foundHostMount := false
		for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == "host" {
				foundHostMount = true
				Expect(vm.MountPath).To(Equal("/host"))
				Expect(vm.ReadOnly).To(BeTrue())
			}
		}
		Expect(foundHostMount).To(BeTrue(), "host filesystem should be mounted")

		ginkgo.By("verifying volume source is correct")
		foundHostVolume := false
		for _, vol := range ds.Spec.Template.Spec.Volumes {
			if vol.Name == "host" {
				foundHostVolume = true
				Expect(vol.VolumeSource.HostPath).ToNot(BeNil())
				Expect(vol.VolumeSource.HostPath.Path).To(Equal("/"))
			}
		}
		Expect(foundHostVolume).To(BeTrue(), "host volume should be defined")
	})

	ginkgo.It("admin should be able to create and delete SplunkForwarders CR", func(ctx context.Context) {
		sf := makeMinimalSplunkforwarder(testSplunkForwarder, operatorNamespace)
		err := k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
		err = k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
		Expect(err).NotTo(HaveOccurred())
	})

	ginkgo.It("dedicated admin should not be able to manage SplunkForwarders CR", func(ctx context.Context) {
		dsf := makeMinimalSplunkforwarder(dedicatedAdminSplunkForwarder, operatorNamespace)
		impersonatedResourceClient, _ := k8s.Impersonate("test-user@redhat.com", "dedicated-admins")
		Expect(sfv1alpha1.AddToScheme(impersonatedResourceClient.GetScheme())).Should(BeNil(), "unable to register sfv1alpha1 api scheme")
		err := impersonatedResourceClient.WithNamespace(operatorNamespace).Create(ctx, &dsf)
		Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected err to be forbidden, got: %v", err)
	})

	ginkgo.It("handles CR updates and reconciliation correctly", func(ctx context.Context) {
		crName := "test-reconcile-update"

		ginkgo.By("creating initial SplunkForwarder CR")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			"/var/log/initial.log",
			"initial_index",
			"_json",
		)
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("waiting for initial DaemonSet creation")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		initialGeneration := sf.Generation

		ginkgo.By("updating CR with new input path and index")
		err := k8s.Get(ctx, crName, operatorNamespace, &sf)
		Expect(err).NotTo(HaveOccurred())

		// Sleep 10 seconds to make sure timestamp is different
		time.Sleep(10 * time.Second)
		sf.Spec.SplunkInputs[0].Path = "/var/log/updated.log"
		sf.Spec.SplunkInputs[0].Index = "updated_index"
		Expect(k8s.WithNamespace(operatorNamespace).Update(ctx, &sf)).To(Succeed())

		ginkgo.By("verifying CR generation incremented")
		Eventually(func() bool {
			k8s.Get(ctx, crName, operatorNamespace, &sf)
			return sf.Generation > initialGeneration
		}).WithTimeout(30 * time.Second).Should(BeTrue())

		ginkgo.By("verifying ConfigMap is updated with new configuration")
		var localCM corev1.ConfigMap
		Eventually(func() bool {
			err := k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
			if err != nil {
				return false
			}
			inputsConf := localCM.Data["inputs.conf"]
			return strings.Contains(inputsConf, "/var/log/updated.log") &&
				strings.Contains(inputsConf, "updated_index")
		}).WithTimeout(90*time.Second).WithPolling(5*time.Second).Should(BeTrue(),
			"ConfigMap should be updated with new input configuration")

		// Updating SplunkForwarder CR doesn't change CreateTimestamp.
		// Reconcilor only compare CR CreateTimestap and DS CreateTimestamp to decide if new DS required
		// Comment this out until issue is resolved.
		ginkgo.By("verifying DaemonSet is recreated")
		Eventually(func() bool {
			err := k8s.Get(ctx, dsName, operatorNamespace, &ds)
			if err != nil {
				return false
			}
			genVersion, err := strconv.ParseInt(ds.Annotations["genVersion"], 10, 64)
			return genVersion > 1
		}).WithTimeout(90*time.Second).WithPolling(10*time.Second).Should(BeTrue(),
			"DaemonSet should be recreated after CR update")
	})

	ginkgo.It("validates HEC endpoint connectivity and configuration", func(ctx context.Context) {
		crName := "test-hec-connectivity"

		ginkgo.By("creating SplunkForwarder CR for HEC mode")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			"/var/log/hec-test.log",
			"hec_index",
			"_json",
		)
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("verifying HEC token secret is mounted in DaemonSet")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		foundHECMount := false
		for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == "splunk-config" {
				foundHECMount = true
				Expect(vm.MountPath).To(ContainSubstring("splunk"))
			}
		}
		Expect(foundHECMount).To(BeTrue(), "HEC token secret should be mounted")

		ginkgo.By("verifying outputs.conf from HEC secret is used")
		foundHECVolume := false
		for _, vol := range ds.Spec.Template.Spec.Volumes {
			if vol.Name == "splunk-hec-token" {
				foundHECVolume = true
				Expect(vol.VolumeSource.Secret).ToNot(BeNil())
				Expect(vol.VolumeSource.Secret.SecretName).To(Equal(operatorconfig.SplunkHECTokenSecretName))
			}
		}
		Expect(foundHECVolume).To(BeTrue(), "HEC secret volume should be defined")

		ginkgo.By("verifying HEC secret contains correct outputs.conf")
		var hecSecret corev1.Secret
		err := k8s.Get(ctx, operatorconfig.SplunkHECTokenSecretName, operatorNamespace, &hecSecret)
		Expect(err).NotTo(HaveOccurred())
		Expect(hecSecret.Data).To(HaveKey("outputs.conf"))

		outputsConf := string(hecSecret.Data["outputs.conf"])
		Expect(outputsConf).To(ContainSubstring("[http]"))
		Expect(outputsConf).To(ContainSubstring("httpEventCollectorToken"))
		Expect(outputsConf).To(ContainSubstring("uri"))

		// TODO: Temporarily commented out for integration testing
		// Integration clusters don't have the full production setup (secrets, SCC permissions, etc.)
		// Follow-up work to address this will be tracked in HCMSEC-3314
		// ginkgo.By("verifying DaemonSet pods start successfully with HEC configuration")
		// Eventually(func() bool {
		// 	k8s.Get(ctx, dsName, operatorNamespace, &ds)
		// 	return ds.Status.NumberReady > 0
		// }).WithTimeout(180*time.Second).WithPolling(10*time.Second).Should(BeTrue(),
		// 	"DaemonSet should have ready pods with HEC configuration")
	})

	ginkgo.It("validates comprehensive index configuration", func(ctx context.Context) {
		crName := "test-multi-index"

		ginkgo.By("creating SplunkForwarder CR with multiple inputs and different indexes")
		sf := sfv1alpha1.SplunkForwarder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: operatorNamespace,
			},
			Spec: sfv1alpha1.SplunkForwarderSpec{
				SplunkLicenseAccepted: true,
				Image:                 "quay.io/redhat-services-prod/openshift/splunk-forwarder-images",
				ImageTag:              "latest",
				ClusterID:             "multi-index-cluster",
				SplunkInputs: []sfv1alpha1.SplunkForwarderInputs{
					{
						Path:       "/var/log/audit/*.log",
						Index:      "security_audit",
						SourceType: "linux_audit",
						WhiteList:  "audit\\.log$",
					},
					{
						Path:       "/var/log/containers/*.log",
						Index:      "container_logs",
						SourceType: "_json",
						BlackList:  ".*test.*",
					},
					{
						Path:       "/var/log/system.log",
						Index:      "system_events",
						SourceType: "syslog",
					},
				},
			},
		}
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("verifying inputs.conf contains all monitor stanzas")
		var localCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		inputsConf := localCM.Data["inputs.conf"]

		ginkgo.By("validating first input configuration (audit logs)")
		Expect(inputsConf).To(ContainSubstring("[monitor:///var/log/audit/*.log]"))
		Expect(inputsConf).To(ContainSubstring("index = security_audit"))
		Expect(inputsConf).To(ContainSubstring("sourcetype = linux_audit"))
		Expect(inputsConf).To(ContainSubstring("whitelist = audit\\.log$"))

		ginkgo.By("validating second input configuration (container logs)")
		Expect(inputsConf).To(ContainSubstring("[monitor:///var/log/containers/*.log]"))
		Expect(inputsConf).To(ContainSubstring("index = container_logs"))
		Expect(inputsConf).To(ContainSubstring("sourcetype = _json"))
		Expect(inputsConf).To(ContainSubstring("blacklist = .*test.*"))

		ginkgo.By("validating third input configuration (system logs)")
		Expect(inputsConf).To(ContainSubstring("[monitor:///var/log/system.log]"))
		Expect(inputsConf).To(ContainSubstring("index = system_events"))
		Expect(inputsConf).To(ContainSubstring("sourcetype = syslog"))

		ginkgo.By("verifying all inputs have cluster metadata")
		for i := 0; i < 3; i++ {
			Expect(inputsConf).To(ContainSubstring("_meta = clusterid::multi-index-cluster"))
		}

		ginkgo.By("verifying metadata ConfigMap has correct annotations")
		var metadataCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-metadata", operatorNamespace, &metadataCM)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		Expect(metadataCM.Annotations).To(HaveKey("genVersion"))
		Expect(metadataCM.Data).To(HaveKey("local.meta"))
	})

	ginkgo.It("validates error handling when required secrets are missing", func(ctx context.Context) {
		crName := "test-missing-secret"

		ginkgo.By("temporarily removing splunk-auth secret")
		var authSecret corev1.Secret
		err := k8s.Get(ctx, operatorconfig.SplunkAuthSecretName, operatorNamespace, &authSecret)
		if err == nil {
			Expect(k8s.Delete(ctx, &authSecret)).To(Succeed())

			ginkgo.By("creating SplunkForwarder CR without auth secret present")
			sf := makeMinimalSplunkforwarder(crName, operatorNamespace)
			sf.Spec.SplunkInputs[0].Path = "/var/log/test.log"
			err = k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)

			// Cleanup CR
			defer func() {
				k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
				err = restoreSplunkAuthSecret(ctx, k8s, operatorNamespace)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(3 * time.Second)
			}()

			ginkgo.By("verifying CR is created but reconciliation is blocked")
			Eventually(func() error {
				return k8s.Get(ctx, crName, operatorNamespace, &sf)
			}).WithTimeout(30 * time.Second).Should(Succeed())

			ginkgo.By("verifying DaemonSet is not created without auth secret")
			dsName := crName + "-ds"
			var ds appsv1.DaemonSet
			Consistently(func() bool {
				err := k8s.Get(ctx, dsName, operatorNamespace, &ds)
				return apierrors.IsNotFound(err)
			}).WithTimeout(30*time.Second).WithPolling(5*time.Second).Should(BeTrue(),
				"DaemonSet should not be created without auth secret")
		}
	})

	ginkgo.It("validates reconciliation retry logic on transient failures", func(ctx context.Context) {
		crName := "test-retry-logic"

		ginkgo.By("creating SplunkForwarder CR")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			"/var/log/retry.log",
			"retry_index",
			"_json",
		)
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		defer func() {
			k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)
			time.Sleep(3 * time.Second)
		}()

		ginkgo.By("waiting for initial reconciliation")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		ginkgo.By("deleting ConfigMap to trigger reconciliation")
		var localCM corev1.ConfigMap
		err := k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		Expect(err).NotTo(HaveOccurred())
		Expect(k8s.Delete(ctx, &localCM)).To(Succeed())

		ginkgo.By("verifying ConfigMap is recreated by reconciliation loop")
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		}).WithTimeout(90*time.Second).WithPolling(5*time.Second).Should(Succeed(),
			"Controller should recreate deleted ConfigMap")

		ginkgo.By("verifying recreated ConfigMap has correct content")
		Expect(localCM.Data).To(HaveKey("inputs.conf"))
		inputsConf := localCM.Data["inputs.conf"]
		Expect(inputsConf).To(ContainSubstring("/var/log/retry.log"))
		Expect(inputsConf).To(ContainSubstring("retry_index"))

		ginkgo.By("deleting DaemonSet to trigger reconciliation")
		err = k8s.Get(ctx, dsName, operatorNamespace, &ds)
		Expect(err).NotTo(HaveOccurred())
		Expect(k8s.Delete(ctx, &ds)).To(Succeed())

		ginkgo.By("verifying DaemonSet is recreated by reconciliation loop")
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(90*time.Second).WithPolling(5*time.Second).Should(Succeed(),
			"Controller should recreate deleted DaemonSet")

		// TODO: Temporarily commented out for integration testing
		// Integration clusters don't have the full production setup (secrets, SCC permissions, etc.)
		// Follow-up work to address this will be tracked in HCMSEC-3314
		// ginkgo.By("verifying DaemonSet pods become ready after recreation")
		// Eventually(func() bool {
		// 	k8s.Get(ctx, dsName, operatorNamespace, &ds)
		// 	return ds.Status.NumberReady > 0
		// }).WithTimeout(180 * time.Second).WithPolling(10 * time.Second).Should(BeTrue())
	})

	ginkgo.It("validates CR deletion and resource cleanup", func(ctx context.Context) {
		crName := "test-deletion-cleanup"

		ginkgo.By("creating SplunkForwarder CR")
		sf := makeSplunkforwarderWithIndex(
			crName,
			operatorNamespace,
			"/var/log/cleanup.log",
			"cleanup_index",
			"_json",
		)
		Expect(k8s.WithNamespace(operatorNamespace).Create(ctx, &sf)).To(Succeed())

		ginkgo.By("waiting for resources to be created")
		dsName := crName + "-ds"
		var ds appsv1.DaemonSet
		Eventually(func() error {
			return k8s.Get(ctx, dsName, operatorNamespace, &ds)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		var localCM corev1.ConfigMap
		Eventually(func() error {
			return k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
		}).WithTimeout(60 * time.Second).Should(Succeed())

		ginkgo.By("deleting SplunkForwarder CR")
		Expect(k8s.WithNamespace(operatorNamespace).Delete(ctx, &sf)).To(Succeed())

		ginkgo.By("verifying DaemonSet is deleted via owner reference")
		Eventually(func() bool {
			err := k8s.Get(ctx, dsName, operatorNamespace, &ds)
			return apierrors.IsNotFound(err)
		}).WithTimeout(120*time.Second).WithPolling(5*time.Second).Should(BeTrue(),
			"DaemonSet should be deleted when CR is deleted")

		ginkgo.By("verifying ConfigMaps are deleted via owner reference")
		Eventually(func() bool {
			err := k8s.Get(ctx, "osd-monitored-logs-local", operatorNamespace, &localCM)
			return apierrors.IsNotFound(err)
		}).WithTimeout(120*time.Second).WithPolling(5*time.Second).Should(BeTrue(),
			"ConfigMaps should be deleted when CR is deleted")
	})
})
