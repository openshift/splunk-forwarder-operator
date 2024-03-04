package config

import (
	"k8s.io/apimachinery/pkg/types"
)

const (
	OperatorName      string = "splunk-forwarder-operator"
	OperatorNamespace string = "openshift-splunk-forwarder-operator"

	SplunkAuthSecretName string = "splunk-auth" // #nosec G101 -- This is a false positive
	ProxyConfigMapName   string = "proxy-config"
)

var ProxyName = types.NamespacedName{
	Name: "cluster",
}

var TrustedCABundleName = types.NamespacedName{
	Namespace: "openshift-config-managed",
	Name:      "trusted-ca-bundle",
}
