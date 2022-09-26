// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	hacommon "github.com/verrazzano/verrazzano/tests/e2e/pkg/ha"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var (
	t           = framework.NewTestFramework("ha-helidon")
	clusterDump = pkg.NewClusterDumpWrapper(namespace)
	clientset   = k8sutil.GetKubernetesClientsetOrDie()

	httpClient *retryablehttp.Client
)

var _ = clusterDump.BeforeSuite(func() {
	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
})

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.Describe("HA Hello Helidon app endpoint test", Label("f:app-lcm.helidon-workload"), func() {

	// GIVEN the hello-helidon app is deployed
	// WHEN we access the app endpoint repeatedly during a Kubernetes cluster upgrade
	// THEN the application endpoint must be accessible during the upgrade
	t.Context("accesses the endpoint", Label("f:mesh.ingress"), func() {
		var host string
		var url string

		t.It("fetches the ingress", func() {
			Eventually(func() (string, error) {
				var err error
				host, err = k8sutil.GetHostnameFromGateway(namespace, "")
				return host, err
			}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))

			url = fmt.Sprintf("https://%s/greet", host)
		})

		hacommon.RunningUntilShutdownIt(t, "accesses /greet app URL", clientset, true, func() {
			Expect(appEndpointAccessible(url, host)).Should(BeTrue())
			time.Sleep(time.Second)
		})
	})
})

// appEndpointAccessible hits the hello-helidon app endpoint and validates that the
// response text matches the expected text
func appEndpointAccessible(url string, hostname string) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		t.Logs.Errorf("Unexpected error while creating new request=%v", err)
		return false
	}

	req.Host = hostname
	req.Close = true
	resp, err := httpClient.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Logs.Errorf("Unexpected error while making http request=%v", err)
		bodyStr, err := readResponseBody(resp)
		if err != nil {
			t.Logs.Errorf("Unexpected error while marshallling error response=%v", err)
			return false
		}

		t.Logs.Errorf("Error Response=%v", bodyStr)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		t.Logs.Errorf("Unexpected status code=%v", resp.StatusCode)
		return false
	}
	bodyStr, err := readResponseBody(resp)
	if err != nil {
		t.Logs.Errorf("Unexpected error marshallling response=%v", err)
		return false
	}
	if !strings.Contains(bodyStr, "Hello World") {
		t.Logs.Errorf("Unexpected response body=%v", bodyStr)
		return false
	}
	return true
}

// readResponseBody reads the response body bytes and returns it as a string
func readResponseBody(resp *http.Response) (string, error) {
	var body string
	if resp != nil && resp.Body != nil {
		bodyRaw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		body = string(bodyRaw)
	}
	return body, nil
}
