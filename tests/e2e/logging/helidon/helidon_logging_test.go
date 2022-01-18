// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var t = framework.NewTestFramework("helidon")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	t.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace("helidon-logging", nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/logging/helidon/helidon-logging-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/logging/helidon/helidon-logging-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	// undeploy the application here
	start := time.Now()
	t.Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/logging/helidon/helidon-logging-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/logging/helidon/helidon-logging-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Eventually(func() error {
		return pkg.DeleteNamespace("helidon-logging")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-workload"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace = "helidon-logging"
)

var _ = t.Describe("Hello Helidon OAM App test", func() {
	// Verify hello-helidon-workload pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Describe("hello-helidon-workload pod", func() {
		t.It("is running", func() {
			t.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-logging namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.It("Get host from gateway.", func() {
		t.Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(testNamespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("for Ingress.", func() {
		t.It("Access /greet App Url.", func() {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			Expect(err).ShouldNot(HaveOccurred())
			t.Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/greet", host)
				return pkg.GetWebPageWithBasicAuth(url, host, "", "", kubeconfigPath)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Hello World")))
		})
	})

	t.Context("for Logging.", func() {
		indexName := fmt.Sprintf("verrazzano-namespace-%s", testNamespace)
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index for hello-helidon namespace is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index for Logging exists", func() {
			t.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello-helidon-container")
		})
		pkg.Concurrently(
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Elasticsearch index for hello-helidon-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Elasticsearch log record exists", func() {
					t.Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon-appconf",
							"kubernetes.container_name":            "hello-helidon-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for container hello-helidon-container")
				})
			},
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Elasticsearch index for other-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Elasticsearch log record of other-container exists", func() {
					t.Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon-appconf",
							"kubernetes.container_name":            "other-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for other-container")
				})
			},
		)
	})
})

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}
