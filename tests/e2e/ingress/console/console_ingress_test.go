// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingress

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "console-ingress"
)

var t = framework.NewTestFramework("ingress")

var _ = t.BeforeSuite(func() {
	deployApplication()
	beforeSuitePassed = true
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
	undeployApplication()
})

func deployApplication() {
	t.Logs.Info("Deploy test application")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	start := time.Now()
	// Wait for namespace to finish deletion possibly from a prior run.
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Create namespace")
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
		return pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	// Verify cidomain-adminserver and mysql pods are running
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{"mysql", "cidomain-adminserver"})
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployApplication() {
	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/ingress/console/application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/ingress/console/components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for application pods to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning(namespace, []string{"mysql", "cidomain-adminserver"})
		return podsTerminated
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("console-ingress app test", Label("f:app-lcm.oam",
	"f:app-lcm.weblogic-workload"), func() {

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
