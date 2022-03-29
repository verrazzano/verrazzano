// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

var _ = t.Describe("Opensearch Retention Policies Suite", Label("f:observability.logging.es"), func() {
	t.It("System log Retention policy in ISM should match configuration value in VZ CR", func() {
		systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.System)
		policyExists, err := pkg.ISMPolicyExists(pkg.SystemLogIsmPolicyName)
		if err != nil {
			Fail("Error retrieving the retention policy for system logs from VZ CR - " + err.Error())
		}
		Expect(policyExists).To(Equal(true))
		if true {
			minIndexAge, err := pkg.GetRetentionPeriod(pkg.SystemLogIsmPolicyName)
			if err != nil {
				Fail("Error retrieving the retention period for system logs from ISM plugin - " + err.Error())
			}
			Expect(minIndexAge).To(Equal(*systemRetentionPolicy.MinIndexAge))
		} else {
			pkg.Log(pkg.Info, "Retention policy for system logs is disabled in VZ CR")
			policyExists, err := pkg.ISMPolicyExists(pkg.SystemLogIsmPolicyName)
			if err != nil {
				Fail("Error validating if ISM policy exists for system logs - " + err.Error())
			}
			Expect(policyExists).To(Equal(false))
		}
	})
	t.It("Application log Retention policy in ISM should match configuration value in VZ CR", func() {
		applicationRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.Application)
		if err != nil {
			Fail("Error retrieving the retention policy for application logs from VZ CR - " + err.Error())
		}
		if true {
			minIndexAge, err := pkg.GetRetentionPeriod(pkg.ApplicationLogIsmPolicyName)
			if err != nil {
				Fail("Error retrieving the retention period of application logs from ISM plugin - " + err.Error())
			}
			Expect(minIndexAge).To(Equal(*applicationRetentionPolicy.MinIndexAge))
		} else {
			policyExists, err := pkg.ISMPolicyExists(pkg.ApplicationLogIsmPolicyName)
			if err != nil {
				Fail("Error validating if the ISM policy for application logs exists - " + err.Error())
			}
			Expect(policyExists).To(Equal(false))
		}
	})

	t.It("Check no system indices exists older than the retention period specified", func() {
		currentEpochTime := time.Now().Unix()
		systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.System)
		if err != nil {
			Fail("Error getting retention policy for system logs from VZ CR - " + err.Error())
		}
		retentionPeriod, err := pkg.CalculateSeconds(*systemRetentionPolicy.MinIndexAge)
		if err != nil {
			Fail("Error converting retention period for system logs from VZ CR to seconds - " + err.Error())
		}
		// Buffer time added to allow ISM policy time to clean up.
		// Link to documentation.
		oldestAllowedTimestamp := (currentEpochTime - retentionPeriod) * 1000
		olderIndexFound, err := pkg.ContainsIndicesOlderThanRetentionPeriod(pkg.SystemLogIsmPolicyName,
			int64(oldestAllowedTimestamp))
		if err != nil {
			Fail("Error checking if older indices for system logs are present - " + err.Error())
		}
		Expect(olderIndexFound).To(Equal(false))
	})

	t.It("Check no application indices exists older than the retention period specified", func() {
		currentEpochTime := time.Now().Unix()
		systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy(pkg.Application)
		if err != nil {
			Fail("Error getting retention period for application logs from VZ CR - " + err.Error())
		}
		retentionPeriod, err := pkg.CalculateSeconds(*systemRetentionPolicy.MinIndexAge)
		if err != nil {
			Fail("Error converting retention policy for application logs from VZ CR to seconds - " + err.Error())
		}
		oldestAllowedTimestamp := (currentEpochTime - retentionPeriod) * 1000
		applicationDataStreams, err := pkg.GetApplicationDataStreamNames()
		if err != nil {
			Fail("Error getting all data stream names that capture the application logs -" + err.Error())
		}
		for _, applicationDataStream := range applicationDataStreams {
			oldIndexFound, err := pkg.ContainsIndicesOlderThanRetentionPeriod(applicationDataStream, int64(oldestAllowedTimestamp))
			if err != nil {
				Fail("Error checking if older indices are present." + err.Error())
			}
			Expect(oldIndexFound).To(Equal(false))
		}
	})

})
