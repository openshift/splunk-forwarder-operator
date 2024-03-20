package matchers

import (
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// BeAvailable is a custom gomega matcher to match on a deployment to be available
//
//	Expect(deployment).Should(BeAvailable())
func BeAvailable() types.GomegaMatcher {
	return gcustom.MakeMatcher(func(deployment *appsv1.Deployment) (bool, error) {
		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable {
				return cond.Status == corev1.ConditionTrue, nil
			}
		}
		return false, nil
	})
}
