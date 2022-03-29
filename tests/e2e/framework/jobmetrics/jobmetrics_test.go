// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jobmetrics

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/util/logs"
	"os"
	"time"

	. "github.com/onsi/gomega"
)

const (
	longWaitTimeout     = 10 * time.Minute
	longPollingInterval = 20 * time.Second
)

var t = framework.NewTestFramework("jobmetrics")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Emit job metrics,", func() {
	t.Context("application Deployment.", func() {
		t.It("at the end of each job", func() {
			log := logs.NewLogger("default")
			log.Print("TEST HERE-----==============")
			t.Logs.Info("HIT THIS TEST=============")
			t.Metrics = t.Metrics.With("job_duration", os.Getenv("DURATION")).
				With("job_status", os.Getenv("JOB_STATUS"))
			Eventually(func() (*v1.SecretList, error) {
				return pkg.ListSecrets("verrazzano-system")
			}, longWaitTimeout, longPollingInterval).ShouldNot(BeNil())
		})
	})
})
