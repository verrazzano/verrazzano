// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var (
	vz             *v1beta1.Verrazzano
	httpClient     *retryablehttp.Client
	vmiCredentials *pkg.UsernamePassword
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	vz, err = pkg.GetVerrazzanoV1beta1()
	Expect(err).To(Not(HaveOccurred()))

	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	vmiCredentials = pkg.EventuallyGetSystemVMICredentials()
})

var _ = BeforeSuite(beforeSuite)

var _ = t.AfterEach(func() {})

var _ = t.Describe("VMI", Label("f:infra-lcm", "f:ui.console"), func() {
	const (
		waitTimeout     = 2 * time.Minute
		pollingInterval = 5 * time.Second
	)

	t.BeforeEach(func() {
		// if Keycloak is disabled, we cannot get the credentials needed for basic auth, so skip the test
		keycloak := vz.Status.Components["keycloak"]
		if keycloak == nil || keycloak.State == v1beta1.CompStateDisabled {
			Skip("Keycloak disabled, skipping test")
		}
	})

	// GIVEN a Verrazzano custom resource
	// WHEN  we attempt to access VMI endpoints present in the CR status
	// THEN  we expect an HTTP OK response status code
	t.DescribeTable("Access VMI endpoints",
		func(getURLFromVZStatus func() *string) {
			url := getURLFromVZStatus()
			if url != nil {
				Eventually(func() (int, error) {
					return httpGet(*url)
				}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(Equal(http.StatusOK))
			}
		},
		Entry("Grafana web UI", func() *string { return vz.Status.VerrazzanoInstance.GrafanaURL }),
		Entry("Prometheus web UI", func() *string { return vz.Status.VerrazzanoInstance.PrometheusURL }),
		Entry("OpenSearch", func() *string { return vz.Status.VerrazzanoInstance.OpenSearchURL }),
		Entry("OpenSearch Dashboards web UI", func() *string { return vz.Status.VerrazzanoInstance.OpenSearchDashboardsURL }),
	)
})

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
