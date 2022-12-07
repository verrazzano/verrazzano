// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package s2

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/psr/tests/scenarios/common"
)

const (
	namespace  = "psrtest"
	scenarioID = "ops-s2"

	waitTimeout     = 2 * time.Minute
	pollingInterval = 5 * time.Second
)

var (
	vz             *v1beta1.Verrazzano
	httpClient     *retryablehttp.Client
	vmiCredentials *pkg.UsernamePassword

	kubeconfig string
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	vz, err = pkg.GetVerrazzanoV1beta1()
	Expect(err).To(Not(HaveOccurred()))

	kubeconfig, _ = k8sutil.GetKubeConfigLocation()

	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	vmiCredentials = pkg.EventuallyGetSystemVMICredentials()
})

func sbsFunc() []byte {
	// Start the scenario if necessary
	kubeconfig, _ = k8sutil.GetKubeConfigLocation()
	common.InitScenario(t, log, scenarioID, namespace, kubeconfig, skipStartScenario)
	return []byte{}
}

var _ = SynchronizedBeforeSuite(sbsFunc, func(bytes []byte) {
	beforeSuite()
})

func sasFunc() {
	// Stop the scenario if necessary
	common.StopScenario(t, log, scenarioID, namespace, skipStopScenario)
}

var _ = SynchronizedAfterSuite(func() {}, sasFunc)

var log = vzlog.DefaultLogger()

var _ = t.Describe("ops-s2", Label("f:psr-ops-s2"), func() {
	// GIVEN a Verrazzano installation with a running PSR ops-s2 scenario
	// WHEN  we wish to validate the PSR workers
	// THEN  the scenario pods are running
	t.DescribeTable("Scenario pods are deployed,",
		func(name string, expected bool) {
			Eventually(func() (bool, error) {
				exists, err := pkg.DoesPodExist(namespace, name)
				if exists {
					t.Logs.Infof("Found pod %s/%s", namespace, name)
				}
				return exists, err
			}, waitTimeout, pollingInterval).Should(Equal(expected))
		},
		t.Entry("PSR ops-s2 getlogs-0 pods running", "psr-ops-s2-ops-getlogs-0-ops-getlogs", true),
		t.Entry("PSR ops-s2 getlogs-1 pods running", "psr-ops-s2-ops-getlogs-1-ops-getlogs", true),
		t.Entry("PSR ops-s2 writelogs-1 pods running", "psr-ops-s2-ops-writelogs-2-ops-writelogs", true),
	)

	// GIVEN a Verrazzano installation
	// WHEN  we wish to validate the PSR workers
	// THEN  we can successfully access the prometheus endpoint
	t.DescribeTable("Verify Prometheus Endpoint",
		func(getURLFromVZStatus func() *string) {
			url := getURLFromVZStatus()
			if url != nil {
				Eventually(func() (int, error) {
					return common.HTTPGet(*url, httpClient, vmiCredentials)
				}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(Equal(http.StatusOK))
			}
		},
		Entry("Prometheus", func() *string { return vz.Status.VerrazzanoInstance.PrometheusURL }),
	)

	// GIVEN a Verrazzano installation
	// WHEN  all opensearch PSR workers are running
	// THEN  metrics can be found for all opensearch PSR workers
	t.DescribeTable("Verify Opensearch ops-s2 Worker Metrics",
		func(metricName string) {
			Eventually(func() bool {
				return pkg.MetricsExistInCluster(metricName, common.GetMetricLabels(""), kubeconfig)
			}, waitTimeout, pollingInterval).Should(BeTrue(),
				fmt.Sprintf("No metrics found for %s", metricName))
		},
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsDataCharsTotalMetric), constants.GetLogsDataCharsTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsFailureCountTotalMetric), constants.GetLogsFailureCountTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsFailureLatencyNanosMetric), constants.GetLogsFailureLatencyNanosMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsLoopCountTotalMetric), constants.GetLogsLoopCountTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsSuccessCountTotalMetric), constants.GetLogsSuccessCountTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsSuccessLatencyNanosMetric), constants.GetLogsSuccessLatencyNanosMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsWorkerLastLoopNanosMetric), constants.GetLogsWorkerLastLoopNanosMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsWorkerRunningSecondsTotalMetric), constants.GetLogsWorkerRunningSecondsTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.GetLogsWorkerThreadCountTotalMetric), constants.GetLogsWorkerThreadCountTotalMetric),

		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoggedCharsTotal), constants.WriteLogsLoggedCharsTotal),
		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoggedLinesTotalCountMetric), constants.WriteLogsLoggedLinesTotalCountMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoopCountTotalMetric), constants.WriteLogsLoopCountTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerLastLoopNanosMetric), constants.WriteLogsWorkerLastLoopNanosMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerRunningSecondsTotalMetric), constants.WriteLogsWorkerRunningSecondsTotalMetric),
		Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerThreadCountTotalMetric), constants.WriteLogsWorkerThreadCountTotalMetric),
	)
})
