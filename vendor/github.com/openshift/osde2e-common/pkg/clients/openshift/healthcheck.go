package openshift

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

const (
	osdClusterReadyNamespace = "openshift-monitoring"
	jobNameLoggerKey         = "job_name"
	timeoutLoggerKey         = "timeout"
)

// OSDClusterHealthy waits for the cluster to be in a healthy "ready" state
// by confirming the osd-ready-job finishes successfully
func (c *Client) OSDClusterHealthy(ctx context.Context, jobName, reportDir string, timeout time.Duration) error {
	var job batchv1.Job

	err := c.Get(ctx, jobName, osdClusterReadyNamespace, &job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err = wait.For(func() (bool, error) {
				if err = c.Get(ctx, jobName, osdClusterReadyNamespace, &job); err != nil {
					if apierrors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			}, wait.WithTimeout(10*time.Minute)); err != nil {
				return fmt.Errorf("job %s never found: %w", jobName, err)
			}
		} else {
			return fmt.Errorf("failed to get existing %s job %w", jobName, err)
		}
	}

	c.log.Info("Wait for cluster job to finish", jobNameLoggerKey, jobName, timeoutLoggerKey, timeout)

	err = wait.For(conditions.New(c.Resources).JobCompleted(&job), wait.WithTimeout(timeout))
	if err != nil {
		logs, err := c.GetJobLogs(ctx, jobName, osdClusterReadyNamespace)
		if err != nil {
			return fmt.Errorf("unable to get job logs for %s/%s: %w", osdClusterReadyNamespace, jobName, err)
		}
		if err = os.WriteFile(fmt.Sprintf("%s/%s.log", reportDir, jobName), []byte(logs), os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write job %s logs to file: %w", jobName, err)
		}
		return fmt.Errorf("%s failed to complete in desired time/health checks have failed: %w", jobName, err)
	}

	c.log.Info("Cluster job finished successfully!", jobNameLoggerKey, jobName)

	return nil
}

// HCPClusterHealthy waits for the cluster to be in a health "ready" state
// by confirming nodes are available
func (c *Client) HCPClusterHealthy(ctx context.Context, computeNodes int, timeout time.Duration) error {
	c.log.Info("Wait for hosted control plane cluster to healthy", timeoutLoggerKey, timeout)

	err := wait.For(func() (bool, error) {
		var nodes corev1.NodeList
		err := c.List(ctx, &nodes)
		if err != nil {
			if os.IsTimeout(err) {
				c.log.Error(err, "timeout occurred contacting api server")
				return false, nil
			}
			return false, err
		}

		if len(nodes.Items) == 0 {
			return false, nil
		}

		for _, node := range nodes.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
		}

		return len(nodes.Items) == computeNodes, nil
	}, wait.WithTimeout(timeout))
	if err != nil {
		return fmt.Errorf("hosted control plane cluster health check failed: %w", err)
	}

	c.log.Info("Hosted control plane cluster health check finished successfully!")

	return nil
}
