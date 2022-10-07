// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobmetrics

import (
	"os"
	"strconv"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
)

var t = framework.NewTestFramework("jobmetrics")

var _ = t.AfterEach(func() {})

// Send job metrics data using Jenkins environment variables
var _ = t.Describe("Emit job metrics", func() {
	t.It("at the end of each job", func() {
		jobDuration, err := strconv.Atoi(os.Getenv("DURATION"))
		if err != nil {
			t.Logs.Errorf("Error parsing job duration from environment variable: %v", err)
			Fail(err.Error())
		}
		timeWaiting, err := strconv.Atoi(os.Getenv("TIME_WAITING"))
		if err != nil {
			t.Logs.Errorf("Error parsing time waiting from environment variable: %v", err)
			Fail(err.Error())
		}
		t.Metrics = t.Metrics.With("job_duration_millis", jobDuration).
			With("job_status", os.Getenv("JOB_STATUS")).
			With("job_time_waiting_millis", timeWaiting)
	})
})
