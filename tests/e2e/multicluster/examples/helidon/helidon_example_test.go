// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute

	multiclusterNamespace = "verrazzano-mc"
	testNamespace         = "hello-helidon"

	projectName   = "hello-helidon"
	componentName = "hello-helidon-component"
	workloadName  = "hello-helidon-workload"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = ginkgo.BeforeSuite(func() {
	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/verrazzano-project.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create hello-helidon project resource: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon application resource: %v", err))
	}
})

var _ = ginkgo.Describe("Multi-cluster Verify Hello Helidon", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("Verify the project exists", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterComponent exists", func() {
			gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the VerrazzanoHelidonWorkload does NOT exist", func() {
			gomega.Expect(componentWorkloadExists()).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("Verify the project exists", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterComponent exists", func() {
			gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the VerrazzanoHelidonWorkload exists", func() {
			gomega.Eventually(componentWorkloadExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify expected pods are running", func() {
			gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})
})

var _ = ginkgo.AfterSuite(func() {
	cleanUp()
})

func cleanUp() {
	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon application resource: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon component resources: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/verrazzano-project.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete hello-helidon project resource: %v", err))
	}

	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err.Error()))
	}

	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", testNamespace, err.Error()))
	}
}

func resourceExists(gvr schema.GroupVersionResource, ns string, name string) bool {
	config := pkg.GetKubeConfig()
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not create dynamic client: %v\n", err))
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

	return u != nil && err == nil
}

func projectExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName)
}

func mcComponentExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "multiclustercomponents",
	}
	return resourceExists(gvr, testNamespace, componentName)
}

func componentWorkloadExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.GroupVersion.Group,
		Version:  oamv1alpha1.GroupVersion.Version,
		Resource: "verrazzanohelidonworkloads",
	}
	return resourceExists(gvr, testNamespace, workloadName)
}

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}
