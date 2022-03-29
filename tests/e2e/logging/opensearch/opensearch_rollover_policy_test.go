// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"sort"
	"strconv"
	"time"
)

const (
	BufferTime = 10 * 60 // 10 minutes
)

var _ = t.Describe("Opensearch Rollover Policies Suite", Label("f:observability.logging.es"), func() {
	t.It("System log Rollover policy in ISM should match configuration value in VZ CR", func() {
		// TODO: After rollover policy is integrated to VZ CR, use the value from VZ.
		_, err := pkg.GetVerrazzanoRolloverPolicy(pkg.System)
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.SystemLogIsmPolicyName)
		if err != nil {
			Fail("Error retrieving the rollover policy for system logs from VZ CR - " + err.Error())
		}
		Expect(rolloverPeriod).To(Equal("1d"))
	})
	t.It("Application log Rollover policy in ISM should match configuration value in VZ CR", func() {
		// TODO: After rollover policy is integrated to VZ CR, use the value from VZ.
		_, err := pkg.GetVerrazzanoRolloverPolicy(pkg.Application)
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.ApplicationLogIsmPolicyName)
		if err != nil {
			Fail("Error retrieving the rollover policy for application logs from VZ CR - " + err.Error())
		}
		Expect(rolloverPeriod).To(Equal("1d"))
	})
	t.It("Data Stream for system logs if older than rollover period should be having more than 1 indices (one per rollover period)", func() {
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.ApplicationLogIsmPolicyName)
		if err != nil {
			Fail("Error retrieving the rollover policy for application logs from VZ CR - " + err.Error())
		}
		rolloverPeriodInSeconds, err := pkg.CalculateSeconds(rolloverPeriod)
		if err != nil {
			Fail("Error converting rollover period to seconds - " + err.Error())
		}
		indexMetadataList, err := pkg.GetIndexMetadataForDataStream(pkg.SystemLogIsmPolicyName)
		if err != nil {
			Fail("Error getting index metadata for data stream verrazzano-system")
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
	t.It("Data Streams of application logs which are older than rollover period should be having more than one backend indices (one per rollover period)", func() {
		rolloverPeriod, err := pkg.GetISMRolloverPeriod(pkg.ApplicationLogIsmPolicyName)
		if err != nil {
			Fail("Error retrieving the rollover policy for application logs from VZ CR - " + err.Error())
		}
		rolloverPeriodInSeconds, err := pkg.CalculateSeconds(rolloverPeriod)
		if err != nil {
			Fail("Error converting rollover period to seconds - " + err.Error())
		}
		applicationDataStreams, err := pkg.GetApplicationDataStreamNames()
		if err != nil {
			Fail("Error getting all data stream names that capture the application logs -" + err.Error())
		}
		for _, applicationDataStream := range applicationDataStreams {
			indexMetadataList, err := pkg.GetIndexMetadataForDataStream(applicationDataStream)
			if err != nil {
				Fail("Error getting index metadata for data stream " + applicationDataStream)
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
