package kube

const (
	MaxEventSize = 100 * 1024 // 100KB per k8s.io/kubernetes/apiserver/pkg/server/options/audit.go
)
