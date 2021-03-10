// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var _ = ginkgo.BeforeSuite(func() {
	if _, err := pkg.CreateNamespace("helidon-logging", map[string]string{"verrazzano-managed": "true"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	if err := pkg.CreateOrUpdateResourceFromFile("testdata/logging/helidon/helidon-logging-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create helidon-logging component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/logging/helidon/helidon-logging-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create helidon-logging application resource: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/logging/helidon/helidon-logging-logging-scope.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create helidon-logging application resource: %v", err))
	}

})

var _ = ginkgo.AfterSuite(func() {
	// undeploy the application here
	err := pkg.DeleteResourceFromFile("testdata/logging/helidon/helidon-logging-app.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-logging application resource: %v\n", err.Error()))
	}
	err = pkg.DeleteResourceFromFile("testdata/logging/helidon/helidon-logging-comp.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-logging component resource: %v\n", err.Error()))
	}
	err = pkg.DeleteNamespace("helidon-logging")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-logging namespace: %v\n", err.Error()))
	}
})

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-workload"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace = "helidon-logging"
)

var _ = ginkgo.Describe("Verify Hello Helidon OAM App.", func() {
	// Verify hello-helidon-workload pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Describe("Verify hello-helidon-workload pod is running.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	var host = ""
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-logging namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	ginkgo.It("Get host from gateway.", func() {
		gomega.Eventually(func() string {
			host = pkg.GetHostnameFromGateway(testNamespace, "")
			return host
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.Describe("Verify Hello Helidon app is working.", func() {
		ginkgo.It("Access /greet App Url.", func() {
			url := fmt.Sprintf("https://%s/greet", host)
			isEndpointAccessible := func() bool {
				return appEndpointAccessible(url, host)
			}
			gomega.Eventually(isEndpointAccessible, 15*time.Second, 1*time.Second).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Logging.", func() {
		indexNameContainer1 := "helidon-logging-hello-helidon-appconf-hello-helidon-component-hello-helidon-container"
		indexNameContainer2 := "helidon-logging-hello-helidon-appconf-hello-helidon-component-other-container"

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the Elasticsearch index for hello-helidon-container is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index for hello-helidon-container exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexNameContainer1)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for hello-helidon-container")
		})

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the Elasticsearch index for other-container is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index for other-container exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexNameContainer2)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for other-container")
		})

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index for hello-helidon-container
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexNameContainer1, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.applicationconfiguration.namespace": "hello-helidon",
					"oam.applicationconfiguration.name":      "hello-helidon-appconf",
					"oam.container.name":                     "hello-helidon-container"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record for container hello-helidon-container")
		})

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index for other-container
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexNameContainer2, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.applicationconfiguration.namespace": "hello-helidon",
					"oam.applicationconfiguration.name":      "hello-helidon-appconf",
					"oam.container.name":                     "other-container"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record for container other-container")
		})
	})
})

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}

func appEndpointAccessible(url string, hostname string) bool {
	status, webpage := pkg.GetWebPageWithBasicAuth(url, hostname, "", "")
	gomega.Expect(status).To(gomega.Equal(http.StatusOK), fmt.Sprintf("GET %v returns status %v expected 200.", url, status))
	gomega.Expect(strings.Contains(webpage, "Hello World")).To(gomega.Equal(true), fmt.Sprintf("Webpage is NOT Hello World %v", webpage))
	return true
}
