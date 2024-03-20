package assertions

import (
	"context"

	"github.com/onsi/gomega"
	"github.com/openshift/osde2e-common/pkg/clients/openshift"
	appsv1 "k8s.io/api/apps/v1"
)

// EventuallyDeployment is a gomega async assertion that can be used with the
// standard or custom gomega matchers
//
//	EventuallyDeployment(ctx, client, "test", "default").Should(BeAvailable())
func EventuallyDeployment(ctx context.Context, client *openshift.Client, name, namespace string) gomega.AsyncAssertion {
	return gomega.Eventually(ctx, func(ctx context.Context) (*appsv1.Deployment, error) {
		var deployment appsv1.Deployment
		err := client.Get(ctx, name, namespace, &deployment)
		return &deployment, err
	})
}
