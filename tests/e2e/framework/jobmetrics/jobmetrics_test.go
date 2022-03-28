// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobmetrics

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"os"
)

var t = framework.NewTestFramework("jobmetrics")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Emit job metrics", func() {
	t.Context("for the job", func() {
		t.It("at the end of each job", func() {
			t.Metrics.With("job_duration", os.Getenv("DURATION")).
				With("job_status", os.Getenv("JOB_STATUS")).
				Info()
			metrics.Emit(t.Metrics)
		})
	})
})
