package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

func (c *Client) UpgradeOperator(ctx context.Context, name, namespace string) error {
	log := c.log.WithValues("name", name, "namespace", namespace)

	log.Info("Attempting to upgrade operator")

	dynamicClient, err := dynamic.NewForConfig(c.GetConfig())
	if err != nil {
		return fmt.Errorf("failed creating the dynamic client: %w", err)
	}

	var (
		csvs = dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "operators.coreos.com",
			Version:  "v1alpha1",
			Resource: "clusterserviceversions",
		}).Namespace(namespace)
		installplans = dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "operators.coreos.com",
			Version:  "v1alpha1",
			Resource: "installplans",
		}).Namespace(namespace)
		subscriptions = dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "operators.coreos.com",
			Version:  "v1alpha1",
			Resource: "subscriptions",
		}).Namespace(namespace)
	)

	start := time.Now()

	// get the current subscription
	subscription, err := subscriptions.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get subscription %s: %w", name, err)
	}
	source, _, err := unstructured.NestedString(subscription.Object, "spec", "source")
	if err != nil {
		return err
	}
	sourceNamespace, _, err := unstructured.NestedString(subscription.Object, "spec", "sourceNamespace")
	if err != nil {
		return err
	}
	channel, _, err := unstructured.NestedString(subscription.Object, "spec", "channel")
	if err != nil {
		return err
	}

	// remove prefix if existing (splunk-forwarder-operator)
	csvName := strings.TrimPrefix(name, "openshift-")

	// find the csv name that matches the subscription name
	installedCSVs, err := csvs.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list CSVs: %w", err)
	}

	var latestCSV string
	for _, installedCSV := range installedCSVs.Items {
		phase, _, err := unstructured.NestedString(installedCSV.Object, "status", "phase")
		if err != nil {
			return err
		}
		if strings.HasPrefix(installedCSV.GetName(), csvName) && phase == "Succeeded" {
			latestCSV = installedCSV.GetName()
		}
	}
	if len(latestCSV) == 0 {
		return fmt.Errorf("failed to find an installed CSV for %q in %q", csvName, namespace)
	}

	replacesCSV, err := c.getReplacesCSV(ctx, csvName, namespace, channel)
	if err != nil {
		return fmt.Errorf("unable to find N-1 CSV for %s in channel %s: %w", name, channel, err)
	}

	log.Info("Uninstalling operator", "version", latestCSV)
	if err = subscriptions.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	if err = csvs.Delete(ctx, latestCSV, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete CSV: %w", err)
	}

	// wait until there is no install plan matching the name
	if err = wait.For(func() (bool, error) {
		ips, err := installplans.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, installplan := range ips.Items {
			ipCSVs, _, err := unstructured.NestedStringSlice(installplan.Object, "spec", "clusterServiceVersionNames")
			if err != nil {
				return false, err
			}
			for _, ipCSV := range ipCSVs {
				if ipCSV == latestCSV {
					return false, nil
				}
			}
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("failed waiting for CSV to be uninstalled: %w", err)
	}

	log.Info("Reinstalling operator", "version", replacesCSV)
	newSubscriptionObject := new(unstructured.Unstructured)
	newSubscriptionObject.SetUnstructuredContent(map[string]any{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "Subscription",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"name":                csvName,
			"source":              source,
			"sourceNamespace":     sourceNamespace,
			"channel":             channel,
			"installPlanApproval": "Automatic",
			"startingCSV":         replacesCSV,
		},
	})
	if _, err = subscriptions.Create(ctx, newSubscriptionObject, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("unable to create new subscription %s: %w", name, err)
	}

	// wait for the install to succeed and that it is upgraded to the originally installed version
	if err = wait.For(func() (bool, error) {
		newSub, err := subscriptions.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		currentCSV, _, err := unstructured.NestedString(newSub.Object, "status", "currentCSV")
		if err != nil {
			return false, err
		}
		return currentCSV == latestCSV, nil
	}); err != nil {
		return fmt.Errorf("failed waiting for CSV %s to be installed at latest version: %w", name, err)
	}

	log.Info("Operator upgrade complete", "duration", time.Since(start), "replaces", replacesCSV, "latest", latestCSV)

	return nil
}

func (c *Client) getReplacesCSV(ctx context.Context, name, namespace, channel string) (string, error) {
	// TODO: the most dependable source of the registry address would be the catalogsource
	var serviceList corev1.ServiceList
	if err := c.WithNamespace(namespace).List(ctx, &serviceList); err != nil {
		return "", fmt.Errorf("unable to list services in %s: %w", namespace, err)
	}
	var service corev1.Service
	for _, svc := range serviceList.Items {
		if strings.HasSuffix(svc.GetName(), "catalog") || strings.HasSuffix(svc.GetName(), "registry") {
			service = svc
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "osde2e-csv-query-",
			Namespace:    namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32(0),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name:         "workdir",
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "install",
							Image:   "registry.access.redhat.com/ubi8/ubi:latest",
							Command: []string{"/bin/sh"},
							Args: []string{
								"-c",
								"curl -vL https://github.com/fullstorydev/grpcurl/releases/download/v${GRPCURL_VERSION}/grpcurl_${GRPCURL_VERSION}_linux_x86_64.tar.gz  | tar -C /workdir/ -zxf - grpcurl",
							},
							Env:          []corev1.EnvVar{{Name: "GRPCURL_VERSION", Value: "1.8.7"}},
							VolumeMounts: []corev1.VolumeMount{{Name: "workdir", MountPath: "/workdir"}},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "grpcurl",
							Image: "registry.access.redhat.com/ubi8/ubi:latest",
							Command: []string{
								"/workdir/grpcurl",
								"-plaintext",
								"-d", fmt.Sprintf(`{"pkgName":%q,"channelName":%q}`, name, channel),
								fmt.Sprintf("%s:50051", service.GetName()),
								"api.Registry/GetBundleForChannel",
							},
							VolumeMounts: []corev1.VolumeMount{{Name: "workdir", MountPath: "/workdir"}},
						},
					},
				},
			},
		},
	}

	if err := c.Create(ctx, job); err != nil {
		return "", fmt.Errorf("failed to create CSV query job in %s namespace: %w", namespace, err)
	}

	defer func() {
		_ = c.Delete(ctx, job, resources.WithDeletePropagation("Background"))
	}()

	// TODO: this doesn't fail _when_ the job fails, it waits for it to be completed
	if err := wait.For(conditions.New(c.Resources).JobCompleted(job), wait.WithTimeout(30*time.Second), wait.WithInterval(3*time.Second)); err != nil {
		// TODO: query pod logs for why we failed
		return "", fmt.Errorf("job %s/%s did not finish successfully: %w", namespace, job.GetName(), err)
	}

	logs, err := c.GetJobLogs(ctx, job.GetName(), namespace)
	if err != nil {
		return "", fmt.Errorf("unable to get job %s/%s logs: %w", namespace, job.GetName(), err)
	}

	output := make(map[string]string)
	if err = json.Unmarshal([]byte(logs), &output); err != nil {
		return "", fmt.Errorf("failed to unmarshal CSV query output: %w", err)
	}

	data := make(map[string]any)
	if err = json.Unmarshal([]byte(output["csvJson"]), &data); err != nil {
		return "", fmt.Errorf("unable to unmarshal csvJson from CSV query: %w", err)
	}

	return data["spec"].(map[string]any)["replaces"].(string), nil
}
