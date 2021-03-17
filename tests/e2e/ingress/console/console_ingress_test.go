// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingress

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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

	pkg.Log(pkg.Info, "Create namespace")
	nsLabels := map[string]string{
		"verrazzano-managed": "true",
		"istio-injection":    "enabled"}
	if _, err := pkg.CreateNamespace(namespace, nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	pkg.Log(pkg.Info, "Create Docker repository secret")
	if _, err := pkg.CreateDockerSecret(namespace, "tododomain-repo-credentials", regServ, regUser, regPass); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Docker registry secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	if _, err := pkg.CreateCredentialsSecret(namespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create database credentials secret")
	if _, err := pkg.CreateCredentialsSecret(namespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create JDBC credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create encryption credentials secret")
	if _, err := pkg.CreatePasswordSecret(namespace, "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create encryption secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create component resources")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/components.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create application resources")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/ingress/console/application.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create application resource: %v", err))
	}
}

func undeployApplication() {
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("testdata/ingress/console/application.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("testdata/ingress/console/components.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete components: %v", err))
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(namespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(namespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

var _ = ginkgo.Describe("Verify application.", func() {

	ginkgo.Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		ginkgo.It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{"mysql", "tododomain-adminserver"})
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
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/console/login/LoginForm.jsp", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Oracle WebLogic Server Administration Console")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		ginkgo.It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("["), pkg.ContainContent("]")))
		})
	})

})
