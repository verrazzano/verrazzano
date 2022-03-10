// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/framework"
	"os"
	"time"
)

// Add function that sums two integers
func Add(a, b int) int {
	return a + b
}

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
)

var kubeConfig = os.Getenv("TEST_KUBECONFIG")
var f = framework.NewDefaultFrameworkWithKubeConfig("add", kubeConfig)

// Label integration to allow --label-filter to filter the tests based on the label using a query
// We might need to consider only the labels starting with f: for the dashboard
var _ = f.Describe("This is a top level describe", Label("integration", "f:app-lcm.oam"), func() {

	BeforeEach(func() {
		// Create the namespace, deploy the application
	})

	AfterEach(func() {
		// Clean test
	})

	f.It("This is the second level, spec", func() {
		fmt.Println("Start running the test spec")
		f.By("adds two numbers together to form a sum")
		gomega.Expect(Add(2, 2) == 4).To(gomega.BeTrue())

		f.By("By with function", func() {
			gomega.Expect(Add(2, 2)).To(gomega.Equal(4))
		})

		f.By("By with function and Eventually")
		gomega.Eventually(func() bool {
			return true
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
		fmt.Println("End of running the test spec")
	})
	framework.Emit(f.Metrics.With(framework.Duration, framework.DurationMillis()))
})
