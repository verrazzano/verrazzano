// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogicworkload

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	componentsPath       = "testdata/loggingtrait/weblogicworkload/weblogic-logging-components.yaml"
	applicationPath      = "testdata/loggingtrait/weblogicworkload/weblogic-logging-application.yaml"
	applicationPodName   = "tododomain-adminserver"
	configMapName        = "logging-stdout-todo-domain-domain"
	namespace            = "weblogic-logging-trait"
)

var kubeConfig = os.Getenv("KUBECONFIG")

var (
	t = framework.NewTestFramework("weblogicworkload")
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	deployWebLogicApplication()
	beforeSuitePassed = true
})

var _ = BeforeSuite(beforeSuite)

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var afterSuite = t.AfterSuiteFunc(func() {
	if failed || !beforeSuitePassed {
		dump.ExecuteBugReport(namespace)
	}
	//loggingtrait.UndeployApplication(namespace, componentsPath, applicationPath, configMapName, t)
})

var _ = AfterSuite(afterSuite)

func deployWebLogicApplication() {
	t.Logs.Info("Deploy test application")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	t.Logs.Info("Create namespace")
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create encryption credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreatePasswordSecret(namespace, "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(componentsPath)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(applicationPath)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Check application pods are running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{"mysql", applicationPodName})
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("Test WebLogic loggingtrait application", Label("f:app-lcm.oam",
	"f:app-lcm.weblogic-workload",
	"f:app-lcm.logging-trait"), func() {

	t.Context("for LoggingTrait.", func() {
		// GIVEN the app is deployed and the pods are running
		// WHEN the app pod is inspected
		// THEN the container for the logging trait should exist
		t.It("Verify that 'logging-stdout' container exists in the 'tododomain-adminserver' pod", func() {
			Eventually(func() bool {
				containerExists, err := pkg.DoesLoggingSidecarExist(kubeConfig, types.NamespacedName{Name: applicationPodName, Namespace: namespace}, "logging-stdout")
				return containerExists && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})

		// GIVEN the app is deployed and the pods are running
		// WHEN the configmaps in the app namespace are retrieved
		// THEN the configmap for the logging trait should exist
		t.It("Verify that 'logging-stdout-tododomain-domain' ConfigMap exists in the 'weblogic-logging-trait' namespace", func() {
			Eventually(func() bool {
				configMap, err := pkg.GetConfigMap(configMapName, namespace)
				return (configMap != nil) && (err == nil)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	t.Context("Ingress.", Label("f:mesh.ingress",
		"f:ui.console"), func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the test namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		t.BeforeEach(func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGateway(namespace, "")
				return host, err
			}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the console endpoint is working.
		// GIVEN the app is deployed
		// WHEN the console endpoint is accessed
		// THEN the expected results should be returned
		t.It("Verify '/console' endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/console/login/LoginForm.jsp", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Oracle WebLogic Server Administration Console")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		t.It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("["), pkg.BodyContains("]")))
		})
	})
})
