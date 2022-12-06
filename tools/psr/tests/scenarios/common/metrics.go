// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
)

// httpGet issues an HTTP GET request with basic auth to the specified URL. httpGet returns the HTTP status code
// and an error.
func httpGet(url string, httpClient *retryablehttp.Client, credentials *pkg.UsernamePassword) (int, error) {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(credentials.Username, credentials.Password)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp.StatusCode, nil
}

// CheckPrometheusEndpoint Verifies the Prometheus endpoint is available in the target cluster
func CheckPrometheusEndpoint(t *framework.TestFramework) {
	vz, err := pkg.GetVerrazzanoV1beta1()
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	credentials := pkg.EventuallyGetSystemVMICredentials()

	// GIVEN a Verrazzano installation
	// WHEN  we wish to validate the PSR workers
	// THEN  we can successfully access the prometheus endpoint
	t.DescribeTable("Verify Prometheus Endpoint",
		func(getURLFromVZStatus func() *string) {
			url := getURLFromVZStatus()
			if url != nil {
				gomega.Eventually(func() (int, error) {
					return httpGet(*url, httpClient, credentials)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.WaitTimeout).Should(gomega.Equal(http.StatusOK))
			}
		},
		ginkgo.Entry("Prometheus", func() *string { return vz.Status.VerrazzanoInstance.PrometheusURL }),
	)
}

// CheckScenarioMetricsExist Verifies the Prometheus endpoint is available in the target cluster and verifies that the specified
// scenario metrics exist in it
func CheckScenarioMetricsExist(t *framework.TestFramework, metrics []string, kubeconfigPath string) {
	// Verify the Prometheus endpoint is available
	CheckPrometheusEndpoint(t)

	testfunc := func(getMetricName func() string) {
		metricName := getMetricName()
		gomega.Eventually(func() bool {
			return pkg.MetricsExistInCluster(metricName, GetMetricLabels(""), kubeconfigPath)
		}, constants.WaitTimeout, constants.PollingInterval).Should(gomega.BeTrue(),
			fmt.Sprintf("No metrics found for %s", metricName))
	}

	// GIVEN a Verrazzano installation
	// WHEN  all opensearch PSR workers are running
	// THEN  metrics can be found for all opensearch PSR workers
	metricsTestTableEntries := []interface{}{testfunc}
	for i := range metrics {
		osWorkerMetric := metrics[i]
		metricsTestTableEntries = append(metricsTestTableEntries,
			ginkgo.Entry(fmt.Sprintf("Verify metric %s", osWorkerMetric), func() string {
				return osWorkerMetric
			}),
		)
	}
	t.DescribeTable("Verify Opensearch Worker Metrics", metricsTestTableEntries...)
}

func GetMetricLabels(_ string) map[string]string {
	return map[string]string{
		//"app_oam_dev_component": podName,
		"verrazzano_cluster": "local",
	}
}
