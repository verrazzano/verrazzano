// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogiclogging

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "weblogic-logging-trait"
	componentsPath		 = "testdata/loggingtrait/weblogicworkload/weblogic-logging-components.yaml"
	applicationPath		 = "testdata/loggingtrait/weblogicworkload/weblogic-logging-application.yaml"
)

var _ = BeforeSuite(func() {
	deployApplication()
})

var _ = AfterSuite(func() {
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
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create encryption credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreatePasswordSecret(namespace, "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "cidomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

func undeployApplication() {
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}

var _ = Describe("Verify application.", func() {

	Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{"mysql", "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	Context("LoggingTrait.", func() {
	})
})
