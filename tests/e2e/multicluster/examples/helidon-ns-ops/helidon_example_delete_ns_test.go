// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcnshelidon

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"time"
)

const (
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
)

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = ginkgo.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = ginkgo.BeforeSuite(func() {
	// deploy the VerrazzanoProject
	err := examples.DeployHelloHelidonProject(adminKubeconfig)
	if err != nil {
		ginkgo.Fail(err.Error())
	}
	// wait for the namespace to be created on the cluster before deploying app
	gomega.Eventually(func() bool {
		return examples.HelidonNamespaceExists(adminKubeconfig)
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue())

	err = examples.DeployHelloHelidonApp(adminKubeconfig)
	if err != nil {
		ginkgo.Fail(err.Error())
	}
})

var _ = ginkgo.Describe("Multi-cluster verify delete ns of hello-helidon", func() {
	ginkgo.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		ginkgo.It("Has multi cluster resources", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyMCResources(adminKubeconfig, true, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		ginkgo.It("Does not have application placed", func() {
			gomega.Consistently(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false)
			}, consistentlyDuration, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		ginkgo.It("Has multi cluster resources", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyMCResources(managedKubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		ginkgo.It("Has application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Delete resources", func() {
		ginkgo.It("Delete project on admin cluster", func() {
			err := deleteProject(adminKubeconfig)
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		ginkgo.It("Delete test namespace on managed cluster", func() {
			if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig); err != nil {
				ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace in managed cluster: %v\n", examples.TestNamespace, err))
			}
		})

		ginkgo.It("Verify deletion on managed cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Delete test namespace on admin cluster", func() {
			if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
				ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace in admin cluster: %v\n", examples.TestNamespace, err))
			}
		})

		ginkgo.It("Verify deletion on admin cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

	})
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

func deleteProject(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}
