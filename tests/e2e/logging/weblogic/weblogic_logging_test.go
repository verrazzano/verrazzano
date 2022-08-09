// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	pkgweblogic "github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	shortPollingInterval     = 10 * time.Second
	imagePullWaitTimeout     = 30 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

var (
	t                  = framework.NewTestFramework("weblogic-logging")
	generatedNamespace = pkg.GenerateNamespace("weblogic-logging")

	failed            = false
	beforeSuitePassed = false

	expectedPods = []string{"weblogicloggingdomain-adminserver"}
)

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

// Create all of the resources to deploy the application and wait for the pods to run
var _ = t.BeforeSuite(func() {
	start := time.Now()

	t.Logs.Info("Deploy WebLogic logging application")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "weblogicloggingdomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "weblogicloggingdomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create persistent volume claim")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/pvc.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/weblogic-logging-comp.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/weblogic-logging-app.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for image pulls")
	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPods)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for running pods")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPods)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())

	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

// Delete all of the resources to undeploy the application
var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(namespace)
	}
	start := time.Now()

	t.Logs.Info("Delete component resources")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/weblogic-logging-app.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete application resources")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/weblogic-logging-comp.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for application pods to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPods)
		return podsTerminated
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Delete persistent volume claim")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace("testdata/logging/weblogic/pvc.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for namespace to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("WebLogic logging test", Label("f:app-lcm.oam", "f:app-lcm.weblogic-workload"), func() {

	t.Context("WebLogic domain is configured properly.", func() {
		// GIVEN the WebLogic domain resource has been created by the Verrazzano Application Operator
		// WHEN we fetch the domain
		// THEN the logHome in the spec has not been overwritten with a /scratch directory
		t.It("logHome has not been overwritten", func() {
			domain, err := pkgweblogic.GetDomain(namespace, "weblogic-logging-domain")
			Expect(domain, err).ShouldNot(BeNil())

			logHome, _, err := unstructured.NestedString(domain.Object, "spec", "logHome")
			Expect(logHome, err).To(Equal("/mnt/shared/logs"))
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
		// GIVEN a WebLogic application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index")
		})

		// GIVEN a WebLogic application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		const k8sContainerNameKeyword = "kubernetes.container_name.keyword"
		const fluentdStdoutSidecarName = "fluentd-stdout-sidecar"

		pkg.Concurrently(
			func() {
				t.It("Verify recent adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "weblogicloggingdomain",
							"kubernetes.labels.app_oam_dev\\/name":  "weblogicloggingdomain-appconf",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent adminserver log record")
				})
			},
			func() {
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: k8sContainerNameKeyword, Value: fluentdStdoutSidecarName},
								{Key: "messageID", Value: "BEA-"},
								{Key: "serverName", Value: "weblogicloggingdomain-adminserver"},
								{Key: "serverName2", Value: "AdminServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent pattern-matched adminserver log record")
				})
			},
			func() {
				t.It("Verify recent pattern-matched WebLogic Server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: k8sContainerNameKeyword, Value: fluentdStdoutSidecarName},
								{Key: "messageID", Value: "BEA-"},
								{Key: "message", Value: "WebLogic Server"},
								{Key: "subSystem", Value: "WebLogicServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent pattern-matched WebLogic Server log record")
				})
			},
			func() {
				t.It("Verify recent fluentd-stdout-sidecar server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: k8sContainerNameKeyword, Value: fluentdStdoutSidecarName},
								{Key: "wls_log_stream", Value: "server_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent fluentd-stdout-sidecar server log record")
				})
			},
			func() {
				t.It("Verify recent fluentd-stdout-sidecar domain log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: k8sContainerNameKeyword, Value: fluentdStdoutSidecarName},
								{Key: "wls_log_stream", Value: "domain_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent fluentd-stdout-sidecar domain log record")
				})
			},
			func() {
				t.It("Verify recent fluentd-stdout-sidecar nodemanager log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecarName},
								{Key: "wls_log_stream", Value: "server_nodemanager_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent server fluentd-stdout-sidecar nodemanager log record")
				})
			},
		)
	})
})
