// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package change_placement

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managed1Kubeconfig = os.Getenv("MANAGED_KUBECONFIG")
// failed indicates whether any of the tests has failed
var failed = false

var _ = ginkgo.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

// Deploy the example resources to the admin cluster
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

var _ = ginkgo.Describe("Multicluster app placed in managed cluster", func() {
	ginkgo.Context("Admin Cluster without app placement", func() {
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
		// THEN expect that the app is not deployed to the admin cluster - check this for some period of time
		ginkgo.It("Does not have application placed", func() {
			gomega.Consistently(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false)
			}, consistentlyDuration, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Managed Cluster with app placement", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		ginkgo.It("Has multi cluster resources", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyMCResources(managed1Kubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		ginkgo.It("Has application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managed1Kubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Change Placement of app to Admin Cluster", func() {
		ginkgo.It("Deploy change-placement manifests to admin cluster", func() {
			err := examples.DeployChangePlacement(adminKubeconfig)
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		ginkgo.It("MC Resources should be removed from managed cluster", func() {
			gomega.Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyMCResources(managed1Kubeconfig, false, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("App should be removed from managed cluster", func() {
			gomega.Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyHelloHelidonInCluster(managed1Kubeconfig, false, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("App should be placed in admin cluster", func() {
			gomega.Eventually(func() bool {
				// app should be placed in the admin cluster
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	// Ensure that if we change placement again, back to the original managed cluster, everything functions
	// as expected. This is needed because the change of placement to admin cluster and the change of placement to
	// a managed cluster are different, and we want to ensure we test the case where the destination cluster is
	// each of the 2 types - admin and managed
	ginkgo.Context("Return the app to Managed Cluster", func() {
		ginkgo.It("Deploy manifests with placement in managed cluster", func() {
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

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has changed placement from admin back to managed cluster
		// THEN expect that the app is not deployed to the admin cluster
		ginkgo.It("Admin cluster does not have application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		// GIVEN a managed cluster
		// WHEN the multi-cluster example application has changed placement to this managed cluster
		// THEN expect that the app is now deployed to the cluster
		ginkgo.It("Managed cluster again has application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managed1Kubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Delete resources on admin cluster", func() {
		ginkgo.It("Delete all the things", func() {
			err := cleanUp()
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})
		ginkgo.It("Verify deletion on admin cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		ginkgo.It("Verify deletion on managed cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managed1Kubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managed1Kubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err))
	}

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", examples.TestNamespace, err))
	}

	// Wait until the namespace is fully deleted in both clusters, so that we don't interfere with other subsequent
	// tests that may use the examples namespace
	gomega.Eventually(func() bool {
		return !pkg.DoesNamespaceExistInCluster(examples.TestNamespace, managed1Kubeconfig) &&
			!pkg.DoesNamespaceExistInCluster(examples.TestNamespace, adminKubeconfig)
	}, waitTimeout, pollingInterval)

})

func cleanUp() error {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-app.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster change-placement application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-comp.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster change-placement component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}
