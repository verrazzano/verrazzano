// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package s1

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/psr/tests/scenarios/common"
)

const (
	namespace  = "psrtest"
	scenarioID = "ops-s1"
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	// Start the scenario if necessary
	kubeconfig, _ := k8sutil.GetKubeConfigLocation()
	common.InitScenario(t, log, scenarioID, namespace, kubeconfig, skipStopScenario)
})

var afterSuite = t.AfterSuiteFunc(func() {
	common.StopScenario(t, log, scenarioID, namespace, skipStopScenario)
})

var _ = BeforeSuite(beforeSuite)

var _ = AfterSuite(afterSuite)

var log = vzlog.DefaultLogger()

var scenarioPods = [][]string{
	{"PSR ops-s1 writelogs-0 pods running", "psr-ops-s1-ops-writelogs-0-ops-writelogs"},
}

var _ = t.Describe("ops-s1", Label("f:psr-ops-s1"), func() {
	// Verify the Scenario pods are up and running
	common.CheckScenarioPods(t, log, namespace, scenarioPods)

	// Verify the Prometheus endpoint is available
	common.CheckPrometheusEndpoint(t)

	// Check that the metrics exist for the scenario
	kubeconfig, _ := k8sutil.GetKubeConfigLocation()
	common.CheckScenarioMetricsExist(t, constants.GetOpsS1Metrics(), kubeconfig)
})
