// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	multiclusterNamespace = "verrazzano-mc"
	appConfigName         = "todo-appconf"
	shortWaitTimeout      = 10 * time.Minute
	shortPollingInterval  = 10 * time.Second
)

var (
	expectedCompsTodoList = []string{
		"todo-domain",
		"todo-jdbc-config",
		"mysql-initdb-config",
		"todo-mysql-service",
		"todo-mysql-deployment"}
	expectedPodsTodoList = []string{
		"mysql",
		"tododomain-adminserver"}
)

// DeployTodoListProject deploys the sock-shop example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeployTodoListProject(kubeconfigPath string, sourceDir string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create %s project resource: %v", sourceDir, err)
	}
	return nil
}

// TodoListNamespaceExists SockShopExists - returns true if the sock-shop namespace exists in the given cluster
func TodoListNamespaceExists(kubeconfigPath string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeconfigPath)
	return err == nil
}

// DeployTodoListApp deploys the sock-shop example application to the cluster with the given kubeConfigPath
func DeployTodoListApp(kubeconfigPath string, sourceDir string) error {

	pkg.Log(pkg.Info, "Deploy ToDoList example")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	// create Docker repository secret
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create Weblogic credentials secret
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create database credentials secret
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/todo-list-components.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s component resources: %v", sourceDir, err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-todo-list-application.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s application resource: %v", sourceDir, err)
	}
	return nil
}
