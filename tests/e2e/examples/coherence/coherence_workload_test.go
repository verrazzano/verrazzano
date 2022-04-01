// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout         = 10 * time.Minute
	shortPollingInterval     = 10 * time.Second
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	appConfiguration  = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-app.yaml"
	compConfiguration = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-comp.yaml"

	httpGetEndPoint  = "hello/coherence"
	httpPostEndPoint = "hello/postMessage"
)

var (
	t                  = framework.NewTestFramework("coherence")
	generatedNamespace = pkg.GenerateNamespace("hello-coherence")
	expectedPods       = []string{"hello-"}
	host               = ""
	welcomeMessage     = "Hello Coherence"
)

var _ = BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		deployCoherenceApp(namespace)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPods)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}

	t.Logs.Info("Coherence Application - check expected pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPods)
		if err != nil {
			AbortSuite(fmt.Sprintf("Coherence application pod is not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Failed to deploy the Coherence Application")

	var err error
	// Get the host from the Istio gateway resource.
	start := time.Now()
	Eventually(func() (string, error) {
		host, err = k8sutil.GetHostnameFromGateway(namespace, "")
		return host, err
	}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		undeployCoherenceApp()
	}
})

func deployCoherenceApp(namespace string) {
	t.Logs.Info("Deploy Coherence application")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(compConfiguration, namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create component resources for Coherence application").ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(appConfiguration, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

func undeployCoherenceApp() {
	t.Logs.Info("Undeploy Coherence application")
	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(appConfiguration, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete component")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(compConfiguration, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pod to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPods)
		return podsTerminated
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for namespace finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("Validate deployment of VerrazzanoCoherenceWorkload", Label("f:app-lcm.oam", "f:app-lcm.coherence-workload"), func() {

	t.Context("Ingress", Label("f:mesh.ingress"), func() {

		// Verify the application endpoints
		t.It("Verify application endpoints", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/%s", host, httpGetEndPoint)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals(welcomeMessage)))

			msg := "This is test message "
			url := fmt.Sprintf("https://%s/%s", host, httpPostEndPoint)
			for i := 0; i < 3; i++ {
				resp, err := pkg.PostWithHostHeader(url, "application/json", host, strings.NewReader(msg+strconv.Itoa(i)))
				if err != nil {
					Fail(fmt.Sprintf("There is an error posting a message, err: %v", host))
				}
				Expect(resp.StatusCode == http.StatusCreated).To(BeTrue(), fmt.Sprintf("HTTP response code %v is not 201", resp.StatusCode))
			}

			url = fmt.Sprintf("https://%s/%s", host, httpGetEndPoint)
			resp, err := pkg.GetWebPage(url, host)
			if err != nil {
				Fail(fmt.Sprintf("There is an error accessing the application, err: %v", err))
			}
			Expect(resp.StatusCode == http.StatusOK).To(BeTrue(), fmt.Sprintf("HTTP response code %v is not 200", resp.StatusCode))
			for i := 0; i < 3; i++ {
				Expect(strings.Contains(string(resp.Body), msg+strconv.Itoa(i))).To(BeTrue(),
					fmt.Sprintf("The response doesn't contain the expected message %s", msg+strconv.Itoa(i)))
			}
		})
	})
	t.Context("Metrics", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		// Verify Prometheus scraped metrics
		t.It("Retrieve application Prometheus scraped metrics", func() {
			kubeConfig, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Expect(err).To(BeNil(), fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			}
			// Coherence metric fix available only from 1.3.0
			if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", kubeConfig); ok {
				Eventually(func() bool {
					return pkg.MetricsExist("vendor:coherence_service_messages_local", "role", "hello")
				}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			}
		})
	})
})
