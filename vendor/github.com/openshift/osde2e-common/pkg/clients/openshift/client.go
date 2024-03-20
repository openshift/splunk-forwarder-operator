package openshift

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/openshift/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

type Client struct {
	*resources.Resources
	log logr.Logger
}

func New(logger logr.Logger) (*Client, error) {
	return NewFromKubeconfig("", logger)
}

func NewFromKubeconfig(filename string, logger logr.Logger) (*Client, error) {
	cfg, err := conf.New(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}
	client, err := resources.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to created dynamic client: %w", err)
	}
	if err = api.Install(client.GetScheme()); err != nil {
		return nil, fmt.Errorf("unable to register openshift api schemes: %w", err)
	}
	return &Client{client, logger}, nil
}

// Impersonate returns a copy of the client with a new ImpersonationConfig
// established on the underlying client, acting as the provided user
//
//	backplaneUser, _ := oc.Impersonate("test-user@redhat.com", "dedicated-admins")
func (c *Client) Impersonate(user string, groups ...string) (*Client, error) {
	if user != "" {
		// these groups are required for impersonating a user
		groups = append(groups, "system:authenticated", "system:authenticated:oauth")
	}

	client := *c
	newRestConfig := rest.CopyConfig(c.Resources.GetConfig())
	newRestConfig.Impersonate = rest.ImpersonationConfig{UserName: user, Groups: groups}
	newResources, err := resources.New(newRestConfig)
	if err != nil {
		return nil, err
	}
	client.Resources = newResources

	if err = api.Install(client.GetScheme()); err != nil {
		return nil, fmt.Errorf("unable to register openshift api schemes: %w", err)
	}

	return &client, nil
}

// GetPodLogs fetches the logs of a pod's default container
func (c *Client) GetPodLogs(ctx context.Context, name, namespace string) (string, error) {
	clientSet, err := kubernetes.NewForConfig(c.GetConfig())
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	logData, err := clientSet.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{}).DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod %s/%s logs: %w", name, namespace, err)
	}
	return string(logData), nil
}

// GetJobLogs fetches the logs of a job's first container
func (c *Client) GetJobLogs(ctx context.Context, name, namespace string) (string, error) {
	pods := new(corev1.PodList)
	err := c.List(ctx, pods, resources.WithLabelSelector(labels.FormatLabels(map[string]string{"job-name": name})))
	if err != nil {
		return "", fmt.Errorf("failed to list pods for job %s in %s namespace: %w", name, namespace, err)
	}
	// TODO: there may be a case where the first item isn't correct
	return c.GetPodLogs(ctx, pods.Items[0].GetName(), namespace)
}
