package kube

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func findAuditExporterDaemonSets(t *testing.T, data any) []*appsv1.DaemonSet {
	t.Helper()
	var results []*appsv1.DaemonSet
	walkYAML(data, func(node map[string]any) {
		kind, _ := node["kind"].(string)
		if kind != "DaemonSet" {
			return
		}
		meta, _ := node["metadata"].(map[string]any)
		if meta == nil {
			return
		}
		name, _ := meta["name"].(string)
		if name != "audit-exporter" {
			return
		}
		raw, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("failed to marshal DaemonSet node: %v", err)
		}
		var ds appsv1.DaemonSet
		if err := json.Unmarshal(raw, &ds); err != nil {
			t.Fatalf("failed to unmarshal DaemonSet: %v", err)
		}
		results = append(results, &ds)
	})
	return results
}

func walkYAML(node any, fn func(map[string]any)) {
	switch v := node.(type) {
	case map[string]any:
		fn(v)
		for _, val := range v {
			walkYAML(val, fn)
		}
	case []any:
		for _, item := range v {
			walkYAML(item, fn)
		}
	}
}

func TestAuditExporterServiceAccountName(t *testing.T) {
	templateParamRe := regexp.MustCompile(`\$\{\{[^}]+\}\}`)

	templateFiles := []string{
		"hack/olm-registry/olm-artifacts-template.yaml",
		"hack/pko/clusterpackage.yaml",
	}

	for _, templateFile := range templateFiles {
		t.Run(templateFile, func(t *testing.T) {
			path := filepath.Join("..", "..", templateFile)
			raw, err := os.ReadFile(path) // #nosec G304 -- path is a hardcoded test fixture
			if err != nil {
				t.Fatalf("failed to read template %s: %v", templateFile, err)
			}

			cleaned := templateParamRe.ReplaceAll(raw, []byte(`"__PLACEHOLDER__"`))

			jsonBytes, err := k8syaml.ToJSON(cleaned)
			if err != nil {
				t.Fatalf("failed to convert YAML to JSON: %v", err)
			}

			var parsed any
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			daemonsets := findAuditExporterDaemonSets(t, parsed)
			if len(daemonsets) == 0 {
				t.Fatal("no audit-exporter DaemonSet found in template")
			}

			for _, ds := range daemonsets {
				podSpec := ds.Spec.Template.Spec

				if podSpec.ServiceAccountName != "splunk-forwarder-operator" {
					t.Errorf("audit-exporter serviceAccountName = %q, want %q",
						podSpec.ServiceAccountName, "splunk-forwarder-operator")
				}

				if len(podSpec.Tolerations) == 0 {
					t.Error("audit-exporter has no tolerations, needs at least one to schedule on master nodes")
				}

				ns := ds.Namespace
				if ns != "openshift-security" {
					t.Errorf("audit-exporter namespace = %q, want %q", ns, "openshift-security")
				}

				if _, ok := podSpec.NodeSelector["node-role.kubernetes.io/master"]; !ok {
					t.Error("audit-exporter missing node-role.kubernetes.io/master nodeSelector")
				}
			}
		})
	}
}

func TestAuditExporterSecurityContext(t *testing.T) {
	for _, ds := range loadAuditExporterDaemonSets(t) {
		t.Run(ds.name, func(t *testing.T) {
			container := ds.ds.Spec.Template.Spec.Containers[0]
			sc := container.SecurityContext

			if sc == nil {
				t.Fatal("audit-exporter container has no securityContext")
			}
			if sc.Privileged == nil || *sc.Privileged {
				t.Error("audit-exporter must not be privileged")
			}
			if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
				t.Error("audit-exporter must set allowPrivilegeEscalation=false")
			}
			if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
				t.Error("audit-exporter must set readOnlyRootFilesystem=true")
			}
			foundDropAll := false
			if sc.Capabilities != nil {
				for _, cap := range sc.Capabilities.Drop {
					if cap == corev1.Capability("ALL") {
						foundDropAll = true
						break
					}
				}
			}
			if !foundDropAll {
				t.Error("audit-exporter must drop ALL capabilities")
			}
		})
	}
}

func TestAuditExporterVolumeMounts(t *testing.T) {
	for _, ds := range loadAuditExporterDaemonSets(t) {
		t.Run(ds.name, func(t *testing.T) {
			container := ds.ds.Spec.Template.Spec.Containers[0]

			mountPaths := map[string]bool{}
			for _, vm := range container.VolumeMounts {
				mountPaths[vm.MountPath] = vm.ReadOnly
			}

			readOnlyMounts := []string{
				"/var/log/kube-apiserver",
				"/var/log/openshift-apiserver",
				"/var/log/oauth-apiserver",
				"/config",
				"/certs",
			}
			for _, m := range readOnlyMounts {
				ro, ok := mountPaths[m]
				if !ok {
					t.Errorf("missing expected mount %s", m)
				} else if !ro {
					t.Errorf("mount %s should be readOnly", m)
				}
			}

			if ro, ok := mountPaths["/var/log/osd-audit"]; !ok {
				t.Error("missing writable mount /var/log/osd-audit")
			} else if ro {
				t.Error("/var/log/osd-audit must be writable (exporter output directory)")
			}
			if ro, ok := mountPaths["/tmp"]; !ok {
				t.Error("missing /tmp mount (needed for readOnlyRootFilesystem)")
			} else if ro {
				t.Error("/tmp must be writable (scratch space for readOnlyRootFilesystem)")
			}
			if _, ok := mountPaths["/var/log"]; ok {
				t.Error("audit-exporter must not mount the entire /var/log")
			}
		})
	}
}

func TestAuditExporterHostPathVolumes(t *testing.T) {
	expectedHostPaths := map[string]string{
		"kube-apiserver-logs":      "/var/log/kube-apiserver",
		"openshift-apiserver-logs": "/var/log/openshift-apiserver",
		"oauth-apiserver-logs":     "/var/log/oauth-apiserver",
		"osd-audit-logs":           "/var/log/osd-audit",
	}

	for _, ds := range loadAuditExporterDaemonSets(t) {
		t.Run(ds.name, func(t *testing.T) {
			hostPaths := map[string]string{}
			for _, vol := range ds.ds.Spec.Template.Spec.Volumes {
				if vol.HostPath != nil {
					hostPaths[vol.Name] = vol.HostPath.Path
				}
			}

			for name, wantPath := range expectedHostPaths {
				gotPath, ok := hostPaths[name]
				if !ok {
					t.Errorf("missing hostPath volume %q", name)
				} else if gotPath != wantPath {
					t.Errorf("volume %q hostPath = %q, want %q", name, gotPath, wantPath)
				}
			}

			for name := range hostPaths {
				if _, allowed := expectedHostPaths[name]; !allowed {
					t.Errorf("unexpected hostPath volume %q not in allow-list", name)
				}
			}
		})
	}
}

type auditExporterDS struct {
	name string
	ds   *appsv1.DaemonSet
}

func loadAuditExporterDaemonSets(t *testing.T) []auditExporterDS {
	t.Helper()
	templateParamRe := regexp.MustCompile(`\$\{\{[^}]+\}\}`)

	templateFiles := []string{
		"hack/olm-registry/olm-artifacts-template.yaml",
		"hack/pko/clusterpackage.yaml",
	}

	results := make([]auditExporterDS, 0, len(templateFiles))
	for _, templateFile := range templateFiles {
		path := filepath.Join("..", "..", templateFile)
		raw, err := os.ReadFile(path) // #nosec G304 -- path is a hardcoded test fixture
		if err != nil {
			t.Fatalf("failed to read template %s: %v", templateFile, err)
		}

		cleaned := templateParamRe.ReplaceAll(raw, []byte(`"__PLACEHOLDER__"`))

		jsonBytes, err := k8syaml.ToJSON(cleaned)
		if err != nil {
			t.Fatalf("failed to convert YAML to JSON: %v", err)
		}

		var parsed any
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("failed to parse template: %v", err)
		}

		for _, ds := range findAuditExporterDaemonSets(t, parsed) {
			results = append(results, auditExporterDS{name: templateFile, ds: ds})
		}
	}

	if len(results) == 0 {
		t.Fatal("no audit-exporter DaemonSets found in templates")
	}
	return results
}

func TestAuditExporterSCCUserMatch(t *testing.T) {
	sccFiles := []string{
		"hack/olm-registry/olm-artifacts-template.yaml",
		"deploy_pko/SecurityContextConstraints-splunkforwarder.yaml",
	}

	for _, sccFile := range sccFiles {
		t.Run(sccFile, func(t *testing.T) {
			path := filepath.Join("..", "..", sccFile)
			raw, err := os.ReadFile(path) // #nosec G304 -- path is a hardcoded test fixture
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			content := string(raw)

			if !strings.Contains(content, "system:serviceaccount:openshift-security:splunk-forwarder-operator") {
				t.Error("SCC must grant access to system:serviceaccount:openshift-security:splunk-forwarder-operator")
			}
		})
	}
}
