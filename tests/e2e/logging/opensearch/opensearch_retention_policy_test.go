// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("Opensearch Retention Policies Suite", Label("f:observability.logging.es"), func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f func()) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.It(description, func() {
				Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			})
		}
		supported, err := pkg.IsVerrazzanoMinVersionEventually("1.3.0", kubeconfigPath)
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

	MinimumVerrazzanoIt("System log Retention policy in ISM should match configuration value in VZ CR", func() {
		Eventually(func() bool {
			systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.SystemLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			policyExists, err := pkg.ISMPolicyExists(systemRetentionPolicy.PolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			minIndexAge, err := pkg.GetRetentionPeriod(systemRetentionPolicy.PolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if systemRetentionPolicy.MinIndexAge == nil {
				pkg.Log(pkg.Error, "MinIndexAge for system ISM policy in VZ CR is nil")
				return false
			}
			return policyExists && minIndexAge == *systemRetentionPolicy.MinIndexAge
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue(), "ISM policy for system indices should be created")
	})

	MinimumVerrazzanoIt("Application log Retention policy in ISM should match configuration value in VZ CR", func() {
		Eventually(func() bool {
			applicationRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.ApplicationLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			policyExists, err := pkg.ISMPolicyExists(applicationRetentionPolicy.PolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			minIndexAge, err := pkg.GetRetentionPeriod(applicationRetentionPolicy.PolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if applicationRetentionPolicy.MinIndexAge == nil {
				pkg.Log(pkg.Error, "MinIndexAge for application ISM policy in VZ CR is nil")
				return false
			}
			return policyExists && minIndexAge == *applicationRetentionPolicy.MinIndexAge
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue(), "ISM policy for application indices should be created")
	})

	MinimumVerrazzanoIt("Check no system indices exists older than the retention period specified", func() {
		currentEpochTime := time.Now().Unix()
		systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.SystemLogIsmPolicyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		retentionPeriod, err := pkg.CalculateSeconds(*systemRetentionPolicy.MinIndexAge)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		// Buffer time added to allow ISM policy time to clean up.
		// Link to documentation.
		oldestAllowedTimestamp := (currentEpochTime - retentionPeriod) * 1000
		indexMetadataList, err := pkg.GetBackingIndicesForDataStream(pkg.SystemLogIsmPolicyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		olderIndexFound, err := pkg.ContainsIndicesOlderThanRetentionPeriod(indexMetadataList, oldestAllowedTimestamp)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		Expect(olderIndexFound).To(Equal(false))
	})

	MinimumVerrazzanoIt("Check no application indices exists older than the retention period specified", func() {
		currentEpochTime := time.Now().Unix()
		applicationRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.ApplicationLogIsmPolicyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		retentionPeriod, err := pkg.CalculateSeconds(*applicationRetentionPolicy.MinIndexAge)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		oldestAllowedTimestamp := (currentEpochTime - retentionPeriod) * 1000
		applicationDataStreams, err := pkg.GetApplicationDataStreamNames()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		var indexMetadataList []pkg.IndexMetadata
		for _, applicationDataStream := range applicationDataStreams {
			indicesPerDataStream, err := pkg.GetBackingIndicesForDataStream(applicationDataStream)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				Fail(err.Error())
			}
			indexMetadataList = append(indexMetadataList, indicesPerDataStream...)
		}
		oldIndexFound, err := pkg.ContainsIndicesOlderThanRetentionPeriod(indexMetadataList, oldestAllowedTimestamp)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		Expect(oldIndexFound).To(Equal(false))

	})

})
