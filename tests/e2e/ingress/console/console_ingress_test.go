// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingress

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "console-ingress"
)

var _ = ginkgo.BeforeSuite(func() {
	deployApplication()
})

var _ = ginkgo.AfterSuite(func() {
	undeployApplication()
})

func deployApplication() {
	pkg.Log(pkg.Info, "Deploy test application")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	// Wait for namespace to finish deletion possibly from a prior run.
	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create Docker repository secret")
	gomega.Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	gomega.Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create database credentials secret")
	gomega.Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create encryption credentials secret")
	gomega.Eventually(func() (*v1.Secret, error) {
		return pkg.CreatePasswordSecret(namespace, "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())
}

func undeployApplication() {
	pkg.Log(pkg.Info, "Delete application")
	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/ingress/console/application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/ingress/console/components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	gomega.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
}

var _ = ginkgo.Describe("Verify application.", func() {

	ginkgo.Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		ginkgo.It("Verify 'cidomain-adminserver' and 'mysql' pods are running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{"mysql", "cidomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Ingress.", func() {
		var host = ""
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the test namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		ginkgo.It("Get host from gateway.", func() {
			gomega.Eventually(func() string {
				host = pkg.GetHostnameFromGateway(namespace, "")
				return host
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()))
		})

		// Verify the console endpoint is working.
		// GIVEN the app is deployed
		// WHEN the console endpoint is accessed
		// THEN the expected results should be returned
		ginkgo.It("Verify '/console' endpoint is working.", func() {
			gomega.Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/console/login/LoginForm.jsp", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(200), pkg.BodyContains("Oracle WebLogic Server Administration Console")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		ginkgo.It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			gomega.Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(200), pkg.BodyContains("["), pkg.BodyContains("]")))
		})
	})
})
