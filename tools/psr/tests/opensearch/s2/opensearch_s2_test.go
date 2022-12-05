// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package s2

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/psrctlcli"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/secrets"
	"io"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longWaitTimeout     = 15 * time.Minute
	longPollingInterval = 30 * time.Second

	namespace  = "psrtest"
	scenarioID = "ops-s2"
)

var (
	vz             *v1beta1.Verrazzano
	httpClient     *retryablehttp.Client
	vmiCredentials *pkg.UsernamePassword

	kubeconfig string

	namespaceLabels = map[string]string{
		"istio-injection":    "enabled",
		"verrazzano-managed": "true",
	}
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	vz, err = pkg.GetVerrazzanoV1beta1()
	Expect(err).To(Not(HaveOccurred()))

	kubeconfig, _ = k8sutil.GetKubeConfigLocation()

	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	vmiCredentials = pkg.EventuallyGetSystemVMICredentials()

	if _, err = pkg.CreateOrUpdateNamespace(namespace, namespaceLabels, nil); err != nil {
		t.Fail(fmt.Sprintf("Error creating or updating namespace %s: %s", namespace, err.Error()))
		return
	}

	if err := secrets.CreateOrUpdatePipelineImagePullSecret(log, namespace, kubeconfig); err != nil {
		t.Fail(fmt.Sprintf("Error creating creating image pull secret for tests suite: %s", err.Error()))
		return
	}

	if !psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		_, stderr, err := psrctlcli.StartScenario(log, scenarioID, namespace)
		if err != nil {
			t.Fail(fmt.Sprintf("Error starting scenario: %s", err.Error()))
			log.Error(string(stderr))
			return
		}
	}
})

var afterSuite = t.AfterSuiteFunc(func() {
	if psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		_, stderr, err := psrctlcli.StopScenario(log, scenarioID, namespace)
		if err != nil {
			log.Errorf("Error starting scenario: %s", err.Error())
			log.Error(string(stderr))
		}
	}
})

var _ = BeforeSuite(beforeSuite)

var _ = AfterSuite(afterSuite)

var log = vzlog.DefaultLogger()

var _ = t.Describe("ops-s2", Label("f:psr-ops-s2"), func() {
	const (
		waitTimeout     = 2 * time.Minute
		pollingInterval = 5 * time.Second
	)

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
					return httpGet(*url)
				}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(Equal(http.StatusOK))
			}
		},
		Entry("Prometheus", func() *string { return vz.Status.VerrazzanoInstance.PrometheusURL }),
	)

	testfunc := func(getMetricName func() string) {
		metricName := getMetricName()
		Eventually(func() bool {
			return pkg.MetricsExistInCluster(metricName, getMetricLabels(""), kubeconfig)
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(),
			fmt.Sprintf("No metrics found for %s", metricName))
	}

	// GIVEN a Verrazzano installation
	// WHEN  all opensearch PSR workers are running
	// THEN  metrics can be found for all opensearch PSR workers
	metrics := constants.GetOpsS2Metrics()
	metricsTestTableEntries := []interface{}{testfunc}
	for i := range metrics {
		osWorkerMetric := metrics[i]
		metricsTestTableEntries = append(metricsTestTableEntries,
			Entry(fmt.Sprintf("Verify metric %s", osWorkerMetric), func() string {
				return osWorkerMetric
			}),
		)
	}
	t.DescribeTable("Verify Opensearch Worker Metrics", metricsTestTableEntries...)
})

func getMetricLabels(_ string) map[string]string {
	return map[string]string{
		//"app_oam_dev_component": podName,
		"verrazzano_cluster": "local",
	}
}

// MetricsRangeExistsInCluster validates the availability of a given metric in the given cluster over an interval
func MetricsRangeExistsInCluster(metricsName string, keyMap map[string]string, kubeconfigPath string) bool {
	end := time.Now()
	start := end.Add(time.Duration(-24) * time.Hour)
	duration := int64(30)
	metric, err := pkg.QueryMetricRange(metricsName, &start, &end, duration, kubeconfigPath)
	if err != nil {
		return false
	}
	metrics := pkg.JTq(metric, "data", "result").([]interface{})
	if metrics != nil {
		return pkg.FindMetric(metrics, keyMap)
	}
	return false
}

// httpGet issues an HTTP GET request with basic auth to the specified URL. httpGet returns the HTTP status code
// and an error.
func httpGet(url string) (int, error) {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(vmiCredentials.Username, vmiCredentials.Password)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp.StatusCode, nil
}
