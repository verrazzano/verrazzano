// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	pollingInterval     = 5 * time.Second
	waitTimeout         = 5 * time.Minute
	longWaitTimeout     = 15 * time.Minute
	longPollingInterval = 20 * time.Second
	sourceDir           = "todo-list"
	testNamespace       = "mc-todo-list"
	testProjectName     = "todo-list"
	testCluster         = "TodoList"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = BeforeSuite(func() {
	// deploy the VerrazzanoProject
	Eventually(func() error {
		return DeployTodoListProject(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return TodoListNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	Eventually(func() error {
		return DeployTodoListApp(adminKubeconfig, sourceDir, testNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
})

var _ = Describe("Multi-cluster verify sock-shop", func() {
	Context("Delete resources", func() {
		It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

func cleanUp(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-todo-list-application.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/todo-list-components.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/tododb-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/weblogic-domain-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/docker-registry-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete sock-shop project resource: %v", err)
	}
	return nil
}
