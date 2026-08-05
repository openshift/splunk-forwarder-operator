package kube

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
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
