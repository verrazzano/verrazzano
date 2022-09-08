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
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var (
	t           = framework.NewTestFramework("ha-helidon")
	clusterDump = pkg.NewClusterDumpWrapper(namespace)
	clientset   = k8sutil.GetKubernetesClientsetOrDie()
)

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.Describe("HA Hello Helidon app test", Label("f:app-lcm.helidon-workload"), func() {
	var host string
	var url string
	var err error

	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the hello-helidon namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			t.Logs.Infof("Host is: %s", host)
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))

		url = fmt.Sprintf("https://%s/greet", host)
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("for Ingress.", Label("f:mesh.ingress"), func() {
		RunningUntilShutdownIt("access /greet app URL", func() {
			Expect(appEndpointAccessible(url, host)).Should(BeTrue())
			time.Sleep(time.Second)
		})
	})
})

func RunningUntilShutdownIt(description string, test func()) {
	t.It(description, func() {
		for {
			test()
			// break out of the loop if we are not running the suite continuously,
			// or the shutdown signal is set
			if ha.IsShutdownSignalSet(clientset) {
				t.Logs.Info("Shutting down...")
				break
			}
		}
	})
}

func appEndpointAccessible(url string, hostname string) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		t.Logs.Errorf("Unexpected error while creating new request=%v", err)
		return false
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Unexpected error while getting kubeconfig location=%v", err)
		return false
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Unexpected error while getting new httpClient=%v", err)
		return false
	}
	req.Host = hostname
	req.Close = true
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Errorf("Unexpected error while making http request=%v", err)
		if resp != nil && resp.Body != nil {
			bodyRaw, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Logs.Errorf("Unexpected error while marshallling error response=%v", err)
				return false
			}

			t.Logs.Errorf("Error Response=%v", string(bodyRaw))
			resp.Body.Close()
		}
		return false
	}

	bodyRaw, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Logs.Errorf("Unexpected error marshallling response=%v", err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		t.Logs.Errorf("Unexpected status code=%v", resp.StatusCode)
		return false
	}
	// HTTP Server headers should never be returned.
	for headerName, headerValues := range resp.Header {
		if strings.EqualFold(headerName, "Server") {
			t.Logs.Errorf("Unexpected Server header=%v", headerValues)
			return false
		}
	}
	bodyStr := string(bodyRaw)
	if !strings.Contains(bodyStr, "Hello World") {
		t.Logs.Errorf("Unexpected response body=%v", bodyStr)
		return false
	}
	return true
}
