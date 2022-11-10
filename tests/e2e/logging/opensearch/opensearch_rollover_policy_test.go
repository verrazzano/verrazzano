// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	BufferTime = 10 * 60 // 10 minutes
)

var _ = t.Describe("Opensearch Rollover Policies Suite", Label("f:observability.logging.es"), func() {
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

	MinimumVerrazzanoIt("System log Rollover policy in ISM should match configuration value in VZ CR", func() {
		Eventually(func() bool {
			rollOverISMPolicy, err := pkg.GetVerrazzanoRolloverPolicy(pkg.SystemLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.SystemLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			return rolloverPeriod == *rollOverISMPolicy.MinIndexAge
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue(), "ISM rollover policy for system logs should match user configured value in VZ")
	})

	MinimumVerrazzanoIt("Application log Rollover policy in ISM should match configuration value in VZ CR", func() {
		Eventually(func() bool {
			rollOverISMPolicy, err := pkg.GetVerrazzanoRolloverPolicy(pkg.ApplicationLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.ApplicationLogIsmPolicyName)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			return rolloverPeriod == *rollOverISMPolicy.MinIndexAge
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue(), "ISM rollover policy for application logs should match user configured value in VZ")
	})

	MinimumVerrazzanoIt("Data Stream for system logs if older than rollover period should be having more than 1 indices (one per rollover period)", func() {
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.SystemLogIsmPolicyName)
		if err != nil {
			Fail(err.Error())
		}
		rolloverPeriodInSeconds, err := pkg.CalculateSeconds(rolloverPeriod)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		indexMetadataList, err := pkg.GetIndexMetadataForDataStream(pkg.SystemLogIsmPolicyName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			Fail(err.Error())
		}
		pkg.Log(pkg.Info, fmt.Sprintf("Data stream %s contains %d backend indices", pkg.SystemLogIsmPolicyName, len(indexMetadataList)))

		if len(indexMetadataList) >= 1 {
			currentTime := time.Now().Unix()
			var sortedCreationTimes []int
			for i := 0; i < len(indexMetadataList); i++ {
				creationTime, _ := strconv.Atoi(indexMetadataList[i].CreationDate)
				pkg.Log(pkg.Info, fmt.Sprintf("Creation time of index %s is %d", indexMetadataList[i].ProvidedName, creationTime))
				sortedCreationTimes = append(sortedCreationTimes, creationTime)
			}
			sort.Ints(sortedCreationTimes)
			for i := 0; i < len(sortedCreationTimes)-1; i++ {
				timeDiffInSeconds := int64((sortedCreationTimes[i+1] - sortedCreationTimes[i]) / 1000)
				isCreateTimesInRange := (timeDiffInSeconds > rolloverPeriodInSeconds) &&
					(timeDiffInSeconds < rolloverPeriodInSeconds+BufferTime)
				Expect(isCreateTimesInRange).To(Equal(true))
			}
			// Check if the last index has not been rolled over and it is less than the rollover period
			currentTimeDiffInSeconds := int64((sortedCreationTimes[len(sortedCreationTimes)-1] / 1000)) - currentTime
			Expect(currentTimeDiffInSeconds < rolloverPeriodInSeconds).To(Equal(true))
		} else {
			Fail("Data stream for system logs should have atleast one backing index")
		}
	})

	MinimumVerrazzanoIt("Data Streams of application logs which are older than rollover period should be having more than one backend indices (one per rollover period)", func() {
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.ApplicationLogIsmPolicyName)
		if err != nil {
			Fail(err.Error())
		}
		rolloverPeriodInSeconds, err := pkg.CalculateSeconds(rolloverPeriod)
		if err != nil {
			Fail(err.Error())
		}
		applicationDataStreams, err := pkg.GetApplicationDataStreamNames()
		if err != nil {
			Fail(err.Error())
		}
		for _, applicationDataStream := range applicationDataStreams {
			indexMetadataList, err := pkg.GetIndexMetadataForDataStream(applicationDataStream)
			if err != nil {
				Fail("Error getting index metadata for application datastream - " + applicationDataStream + ": " + err.Error())
			}
			pkg.Log(pkg.Info, fmt.Sprintf("Data stream %s contains %d backend indices", pkg.SystemLogIsmPolicyName, len(indexMetadataList)))
			if len(indexMetadataList) >= 1 {
				currentTime := time.Now().Unix()
				var sortedCreationTimes []int
				for i := 0; i < len(indexMetadataList); i++ {
					creationTime, _ := strconv.Atoi(indexMetadataList[i].CreationDate)
					pkg.Log(pkg.Info, fmt.Sprintf("Creation time of index %s is %d", indexMetadataList[i].ProvidedName, creationTime))
					sortedCreationTimes = append(sortedCreationTimes, creationTime)
				}
				sort.Ints(sortedCreationTimes)
				for i := 0; i < len(sortedCreationTimes)-1; i++ {
					timeDiffInSeconds := int64((sortedCreationTimes[i+1] - sortedCreationTimes[i]) / 1000)
					isCreateTimesInRange := (timeDiffInSeconds > rolloverPeriodInSeconds) &&
						(timeDiffInSeconds < rolloverPeriodInSeconds+BufferTime)
					Expect(isCreateTimesInRange).To(Equal(true))
				}
				// Check if the last index has not been rolled over and it is less than the rollover period
				currentTimeDiffInSeconds := int64((sortedCreationTimes[len(sortedCreationTimes)-1] / 1000)) - currentTime
				Expect(currentTimeDiffInSeconds < rolloverPeriodInSeconds).To(Equal(true))
			} else {
				Fail("No index present for data stream - " + applicationDataStream)
			}
		}
	})

})
