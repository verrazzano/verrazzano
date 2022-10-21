// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

var (
	t                  = framework.NewTestFramework("helidon")
	generatedNamespace = pkg.GenerateNamespace("helidon-logging")
)

var _ = t.BeforeSuite(func() {
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		file, err := pkg.FindTestDataFile("testdata/logging/helidon/helidon-logging-comp.yaml")
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		file, err := pkg.FindTestDataFile("testdata/logging/helidon/helidon-logging-app.yaml")
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	// Verify hello-helidon-workload pod is running
	Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(namespace)
	}
	// undeploy the application here
	start := time.Now()

	t.Logs.Info("Delete application")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("testdata/logging/helidon/helidon-logging-app.yaml")
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("testdata/logging/helidon/helidon-logging-comp.yaml")
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for application pods to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning("helidon-logging", expectedPodsHelloHelidon)
		return podsTerminated
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved("helidon-logging")
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetNamespace("helidon-logging")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-workload"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

var _ = t.Describe("Hello Helidon OAM App test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload"), func() {

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-logging namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("for Ingress.", Label("f:mesh.ingress"), func() {
		t.It("Access /greet App Url.", func() {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/greet", host)
				response, err := pkg.GetWebPageWithBasicAuth(url, host, "", "", kubeconfigPath)
				//This test is failing intermittently with 403. This is a temporary fix
				//untill a solution is found.
				if response != nil && response.StatusCode == 403 {
					t.Logs.Error("/greet returned 403.")
				}
				return response, err
			}, shortWaitTimeout, shortPollingInterval).Should(Or(And(pkg.HasStatus(200), pkg.BodyContains("Hello World")), pkg.HasStatus(403)))
		})
	})

	t.Context("for Logging.", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index for hello-helidon namespace is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index for Logging exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello-helidon-container")
		})
		pkg.Concurrently(
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Elasticsearch index for hello-helidon-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Elasticsearch log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon",
							"kubernetes.container_name":            "hello-helidon-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for container hello-helidon-container")
				})
			},
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Elasticsearch index for other-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Elasticsearch log record of other-container exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon",
							"kubernetes.container_name":            "other-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for other-container")
				})
			},
		)
	})
})

func helloHelidonPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}
