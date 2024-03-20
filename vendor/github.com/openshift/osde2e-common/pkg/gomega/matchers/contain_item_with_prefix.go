package matchers

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

// ContainItemWithPrefix is a gomega matcher that can be used to assert that a
// Kubernetes list object contains an item name with the provided prefix
//
//	var rolebindings rbacv1.RoleBindingList
//	err = k8s.List(ctx, &rolebindings)
//	Expect(err).ShouldNot(HaveOccurred(), "failed to list rolebindings")
//	Expect(&roleBindingsList).Should(ContainItemWithPrefix("test"))
func ContainItemWithPrefix(prefix string) types.GomegaMatcher {
	return gcustom.MakeMatcher(func(list k8s.ObjectList) (bool, error) {
		items, err := meta.ExtractList(list)
		if err != nil {
			return false, fmt.Errorf("not a list type: %w", err)
		}
		for _, item := range items {
			accessor, err := meta.Accessor(item)
			if err != nil {
				return false, fmt.Errorf("unable to get item's objectmeta: %w", err)
			}
			if strings.HasPrefix(accessor.GetName(), prefix) {
				return true, nil
			}
		}
		return false, nil
	})
}
