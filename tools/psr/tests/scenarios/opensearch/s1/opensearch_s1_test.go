// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package s1

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/psr/tests/scenarios/common"
)

const (
	namespace  = "psrtest"
	scenarioID = "ops-s1"

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

	// Start the scenario if necessary
	common.InitScenario(t, log, scenarioID, namespace, kubeconfig, skipStartScenario)
})

var afterSuite = t.AfterSuiteFunc(func() {
	common.StopScenario(t, log, scenarioID, namespace, skipStopScenario)
})

var _ = BeforeSuite(beforeSuite)

var _ = AfterSuite(afterSuite)

var log = vzlog.DefaultLogger()

var _ = t.Describe("ops-s1", Label("f:psr-ops-s1"), func() {
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
		t.Entry("PSR ops-s1 writelogs-0 pods running", "psr-ops-s1-ops-writelogs-0-ops-writelogs", true),
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
	//t.DescribeTable("Verify Opensearch ops-s1 Worker Metrics",
	//	func(metricName string) {
	//		Eventually(func() bool {
	//			return pkg.MetricsExistInCluster(metricName, common.GetMetricLabels(""), kubeconfig)
	//		}, waitTimeout, pollingInterval).Should(BeTrue(),
	//			fmt.Sprintf("No metrics found for %s", metricName))
	//	},
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoggedCharsTotal), constants.WriteLogsLoggedCharsTotal),
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoggedLinesTotalCountMetric), constants.WriteLogsLoggedLinesTotalCountMetric),
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsLoopCountTotalMetric), constants.WriteLogsLoopCountTotalMetric),
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerLastLoopNanosMetric), constants.WriteLogsWorkerLastLoopNanosMetric),
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerRunningSecondsTotalMetric), constants.WriteLogsWorkerRunningSecondsTotalMetric),
	//	Entry(fmt.Sprintf("Verify metric %s", constants.WriteLogsWorkerThreadCountTotalMetric), constants.WriteLogsWorkerThreadCountTotalMetric),
	//)
})
