package verify

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("After upgrade is complete", Label("f:platform-lcm.upgrade"), func() {

	// GIVEN the verrazzano-system namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	t.It("an install should not begin", func() {
		Eventually(func() bool {
			return upgradeCompleteIsLastCondition()
		}, fiveMinutes, pollingInterval).Should(BeTrue(), "Expected UpgradeComplete to be the last condition")
	})
})

func upgradeCompleteIsLastCondition() bool {
	vz, err := pkg.GetVerrazzano()
	if err != nil {
		t.Logs.Errorf("Error getting Verrazzano CR: %v", err)
		return false
	}
	lastCondtion := vz.Status.Conditions[len(vz.Status.Conditions)-1]
	return lastCondtion.Type == vzapi.CondUpgradeComplete
}
