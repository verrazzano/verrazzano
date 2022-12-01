// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	disableErrorMsg      = "disabling component jaegerOperator is not allowed"
)

var (
	// Initialize the Test Framework
	t     = framework.NewTestFramework("update jaeger operator")
	start = time.Now()
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	m := JaegerOperatorEnabledModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Update Jaeger", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	t.It("traces from verrazzano system components should be available when queried from Jaeger", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Fail(err.Error())
		}
		validatorFn := pkg.ValidateSystemTracesFuncInCluster(kubeconfigPath, start, "local")
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	t.It("traces from verrazzano system components should be available in the OS backend storage.", func() {
		validatorFn := pkg.ValidateSystemTracesInOSFunc(start)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})
})
