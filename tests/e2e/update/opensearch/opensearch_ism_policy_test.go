// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	"time"
)

var _ = t.Describe("Update opensearch ISM policies", Label("f:platform-lcm.update"), func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f func()) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.It(description, func() {
				Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			})
		}
		supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
		if err != nil {
			t.It(description, func() {
				Fail(err.Error())
			})
		}
		// Only run tests if Verrazzano is at least version 1.3.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.3.0", description))
		}
	}

	MinimumVerrazzanoIt("opensearch update system retention policy", func() {
		m := pkg.ElasticSearchISMPolicyAddModifier{}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		// Wait for sufficient time to allow the VMO reconciliation to complete
		pkg.WaitForISMPolicyUpdate(pollingInterval, waitTimeout)
		validateRetentionPolicy(pkg.SystemLogIsmPolicyName, pkg.DefaultRetentionPeriod, pollingInterval, waitTimeout)
	})

	MinimumVerrazzanoIt("opensearch update application retention policy", func() {
		m := pkg.ElasticSearchISMPolicyAddModifier{}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		// Wait for sufficient time to allow the VMO reconciliation to complete
		pkg.WaitForISMPolicyUpdate(pollingInterval, waitTimeout)
		validateRetentionPolicy(pkg.ApplicationLogIsmPolicyName, pkg.DefaultRetentionPeriod, pollingInterval, waitTimeout)
	})

	MinimumVerrazzanoIt("opensearch update system rollover policy", func() {
		m := pkg.ElasticSearchISMPolicyAddModifier{}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		// Wait for sufficient time to allow the VMO reconciliation to complete
		pkg.WaitForISMPolicyUpdate(pollingInterval, waitTimeout)
		validateRolloverPolicy(pkg.SystemLogIsmPolicyName, pkg.DefaultRolloverPeriod, pollingInterval, waitTimeout)
	})

	MinimumVerrazzanoIt("opensearch update application rollover policy", func() {
		m := pkg.ElasticSearchISMPolicyAddModifier{}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		// Wait for sufficient time to allow the VMO reconciliation to complete
		pkg.WaitForISMPolicyUpdate(pollingInterval, waitTimeout)
		validateRolloverPolicy(pkg.ApplicationLogIsmPolicyName, pkg.DefaultRolloverPeriod, pollingInterval, waitTimeout)
	})
})

func validateRetentionPolicy(policyName string, retentionPeriod string, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		policyExists, err := pkg.ISMPolicyExists(policyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		minIndexAge, err := pkg.GetRetentionPeriod(policyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return policyExists && minIndexAge == retentionPeriod
	}).WithPolling(pollingInterval).WithTimeout(timeout)
}

func validateRolloverPolicy(policyName string, expectedRolloverPeriod string, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(policyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return rolloverPeriod == expectedRolloverPeriod
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue(), "ISM rollover policy for application logs should match user configured value in VZ")
}
