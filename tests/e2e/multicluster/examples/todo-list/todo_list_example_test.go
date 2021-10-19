// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"encoding/json"
	"fmt"
	"os"
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
	sourceDir            = "todo-list"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

var _ = BeforeSuite(func() {
	deployToDoListExample()
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	//undeployToDoListExample()
})

func deployToDoListExample() {
	pkg.Log(pkg.Info, "Deploy ToDoList example")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace("todo-list", nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml")
	}, shortWaitTimeout, shortPollingInterval, "Failed to create application resource").ShouldNot(HaveOccurred())
}

func undeployToDoListExample() {
	pkg.Log(pkg.Info, "Undeploy ToDoList example")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace("todo-list")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Deleted namespace check")
	Eventually(func() bool {
		_, err := pkg.GetNamespace("todo-list")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config certificate generated to support secure gateways is fetched
	// THEN the certificate should have been cleaned up
	pkg.Log(pkg.Info, "Deleted certificate check")
	Eventually(func() bool {
		_, err := pkg.GetCertificate("istio-system", "todo-list-todo-appconf-cert")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config secret generated to support secure gateways is fetched
	// THEN the secret should have been cleaned up
	pkg.Log(pkg.Info, "Waiting for secret containing certificate to be deleted")
	var secret *v1.Secret
	var err error
	for i := 0; i < 30; i++ {
		secret, err = pkg.GetSecret("istio-system", "todo-list-todo-appconf-cert-secret")
		if err != nil && errors.IsNotFound(err) {
			pkg.Log(pkg.Info, "Secret deleted")
			return
		}
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error attempting to get secret: %v", err))
		}
		time.Sleep(shortPollingInterval)
	}

	pkg.Log(pkg.Error, "Secret could not be deleted. Secret data:")
	if secret != nil {
		if b, err := json.Marshal(secret); err == nil {
			pkg.Log(pkg.Info, string(b))
		}
	}
	pkg.ExecuteClusterDumpWithEnvVarConfig()
	Fail("Unable to delete secret")
}
