// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute

	multiclusterNamespace = "verrazzano-mc"
	testNamespace         = "hello-helidon"

	projectName   = "hello-helidon"
	appConfigName = "hello-helidon-appconf"
	componentName = "hello-helidon-component"
	workloadName  = "hello-helidon-workload"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managed1Kubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// Deploy the example resources to the admin cluster
var _ = ginkgo.BeforeSuite(func() {
	examples.DeployHelloHelidon(adminKubeconfig)
})

var _ = ginkgo.Describe("Multi-cluster Verify Hello Helidon", func() {
	ginkgo.Context("Admin Cluster without app placement", func() {
		examples.VerifyHelloHelidonInAdminCluster(adminKubeconfig, false)
	})

	ginkgo.Context("Managed Cluster with app placement", func() {
		examples.VerifyHelloHelidonInManagedCluster(managed1Kubeconfig, true)
	})

	ginkgo.Context("Change Placement of app to Admin Cluster", func() {
		ginkgo.It("Deploy change-placement manifests to admin cluster", func() {
			examples.DeployChangePlacement(adminKubeconfig)
		})

		// app should not be placed in the managed cluster
		examples.VerifyHelloHelidonDeletedInCluster(managed1Kubeconfig, false)

		// app should be placed in admin cluster
		examples.VerifyHelloHelidonInAdminCluster(adminKubeconfig, true)
	})

	ginkgo.Context("Delete resources on admin cluster", func() {
		ginkgo.It("Delete all the things", func() {
			cleanUp()
		})

		// verify deletion from admin cluster where the app was placed
		examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, true)
	})
})

var _ = ginkgo.AfterSuite(func() {
	cleanUp()

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managed1Kubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err))
	}

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", examples.TestNamespace, err))
	}
})

func cleanUp() {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-app.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster change-placement application resource: %v", err))
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-comp.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster change-placement component resources: %v", err))
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete hello-helidon project resource: %v", err))
	}
}
