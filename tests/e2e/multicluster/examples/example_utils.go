// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package examples

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
	TestNamespace = "hello-helidon"
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute

	multiclusterNamespace = "verrazzano-mc"
	projectName   = "hello-helidon"
	appConfigName = "hello-helidon-appconf"
	componentName = "hello-helidon-component"
	workloadName  = "hello-helidon-workload"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}

// DeployHelloHelidon deploys the hello-helidon example to the cluster with the given kubeConfigPath
func DeployHelloHelidon(kubeConfigPath string) {
	os.Setenv("TEST_KUBECONFIG", kubeConfigPath)
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", kubeConfigPath); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create hello-helidon project resource: %v", err))
	}
	// wait for the namespace to be created on the admin cluster before applying components and app config
	gomega.Eventually(func() bool {
		return namespaceExists(kubeConfigPath)
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue())

	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml", kubeConfigPath); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml", kubeConfigPath); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon application resource: %v", err))
	}
}

// VerifyHelloHelidonInAdminCluster verifies the hello-helidon application in the given admin cluster
// In an admin cluster, MC resources must be present, but unwrapped resources should only exist if
// placedInAdminCluster = true, and not exist otherwise
func VerifyHelloHelidonInAdminCluster(kubeconfigPath string, placedInAdminCluster bool) {
	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect that the mulit-cluster app config has been created on the admin cluster
	ginkgo.It("Verify the MultiClusterApplicationConfiguration exists", func() {
		gomega.Eventually(func() bool {
			return mcAppConfExists(kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect that the multi-cluster component has been created on the admin cluster
	ginkgo.It("Verify the MultiClusterComponent exists", func() {
		gomega.Eventually(func() bool {
			return mcComponentExists(kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	})

	verifyHelloHelidonInCluster(kubeconfigPath, placedInAdminCluster)
}

// VerifyHelloHelidonInManagedCluster verifies the hello-helidon application in the given managed cluster
// In a managed cluster, MC resources as well as unwrapped resources should only exist if
// placedInThisCluster = true, and not exist otherwise
func VerifyHelloHelidonInManagedCluster(kubeconfigPath string, placedInThisCluster bool) {
	// no mc resources if not placed here
	// GIVEN a managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect that the multi-cluster app config exists on managed cluster ONLY if the app is placed in the cluster
	ginkgo.It("Verify the MultiClusterApplicationConfiguration does not exist on this managed cluster", func() {
		pkg.Log(pkg.Info, "Testing against cluster with kubeconfig: "+kubeconfigPath)
		gomega.Expect(mcAppConfExists(kubeconfigPath)).Should(gomega.Equal(placedInThisCluster))
	})

	// GIVEN a managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect that the multi-cluster component exists on managed cluster ONLY if the app is placed in the cluster
	ginkgo.It("Verify the MultiClusterComponent does not exist on this managed cluster", func() {
		pkg.Log(pkg.Info, "Testing against cluster with kubeconfig: "+kubeconfigPath)
		gomega.Expect(mcComponentExists(kubeconfigPath)).Should(gomega.Equal(placedInThisCluster))
	})

	verifyHelloHelidonInCluster(kubeconfigPath, placedInThisCluster)
}

func verifyHelloHelidonInCluster(kubeConfigPath string, placedInThisCluster bool) {
	// GIVEN an admin or managed cluster
	// WHEN the multi-cluster example application has been created on admin cluster
	// THEN expect that the project has been created on this cluster
	ginkgo.It("Verify the project exists", func() {
		gomega.Eventually(func() bool {
			return projectExists(kubeConfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	})

	appExistsSuffix := "does NOT exist"
	if placedInThisCluster {
		appExistsSuffix = "exists"
	}
	podsRunningTestCaseName := "Verify pods are NOT running"
	if placedInThisCluster {
		podsRunningTestCaseName = "Verify expected pods are running"
	}
	// GIVEN an admin or managed cluster
	// WHEN the multi-cluster example application has been created on admin cluster and placed in this cluster
	// THEN expect that the component workload HAS been unwrapped in this cluster
	// GIVEN an admin or managed cluster
	// WHEN the multi-cluster example application has been created on admin cluster and NOT placed in this cluster
	// THEN expect that the component workload has NOT been unwrapped on the admin cluster
	ginkgo.It(fmt.Sprintf("Verify the VerrazzanoHelidonWorkload %s", appExistsSuffix), func() {
		gomega.Expect(componentWorkloadExists(kubeConfigPath)).Should(gomega.Equal(placedInThisCluster))
	})

	// GIVEN an admin or managed cluster
	// WHEN the multi-cluster example application has been created on admin cluster and placed in this cluster
	// THEN expect that the application pods are running
	// GIVEN an admin or managed cluster
	// WHEN the multi-cluster example application has been created on admin cluster and NOT placed in this cluster
	// THEN expect that the pods are NOT running
	ginkgo.It(podsRunningTestCaseName, func() {
		gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.Equal(placedInThisCluster))
	})
}

func VerifyHelloHelidonDeletedAdminCluster(kubeconfigPath string, placedInAdminCluster bool) {
	// GIVEN an admin cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the multi-cluster app config to no longer exist on the admin cluster
	ginkgo.It("Verify the MultiClusterApplicationConfiguration no longer exists on the admin cluster", func() {
		gomega.Eventually(func() bool {
			return mcAppConfExists(kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	})

	// GIVEN an admin cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the multi-cluster component to no longer exist on the admin cluster
	ginkgo.It("Verify the MultiClusterComponent no longer exists on the admin cluster", func() {
		gomega.Eventually(func() bool {
			return mcComponentExists(kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	})

	VerifyHelloHelidonDeletedInCluster(kubeconfigPath, placedInAdminCluster)
}

func VerifyHelloHelidonDeletedInCluster(kubeconfigPath string, placedInThisCluster bool) {
	// GIVEN an admin or managed cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the project to no longer exist on this cluster
	ginkgo.It("Verify the project no longer exists on the cluster", func() {
		gomega.Eventually(func() bool {
			return projectExists(kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	})

	// NOTE: These tests are disabled pending a fix for VZ-2454 - once fixed, the mcappconf and mccomp
	// deletion checks can be removed from VerifyHelloHelidonDeletedAdminCluster as they are duplicated here

	// GIVEN an admin or managed cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the multi-cluster app config to no longer exist on this cluster
	// ginkgo.It("Verify the MultiClusterApplicationConfiguration no longer exists on the cluster", func() {
	// 	gomega.Eventually(func() bool {
	//  	return mcAppConfExists(kubeconfigPath)
	// 	}, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	// })

	// GIVEN an admin or managed cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the multi-cluster component to no longer exist on this cluster
	// ginkgo.It("Verify the MultiClusterComponent no longer exists on the cluster", func() {
	// 	gomega.Eventually(func() bool {
	//  	return mcComponentExists(kubeconfigPath)
	// 	}, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	// })

	// GIVEN an admin or managed cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the component workload to no longer exist on this cluster
	// ginkgo.It("Verify the VerrazzanoHelidonWorkload no longer exists on the cluster", func() {
	// 	gomega.Eventually(componentWorkloadExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	// })

	// GIVEN an admin or managed cluster
	// WHEN the example application has been deleted from the admin cluster
	// THEN expect the application pods to no longer be running on this cluster
	// ginkgo.It("Verify expected pods are no longer running", func() {
	// 	gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeFalse())
	// })
}

func namespaceExists(kubeconfigPath string) bool {
	_, err := pkg.GetNamespaceInCluster(TestNamespace, kubeconfigPath)
	return err == nil
}

func projectExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName, kubeconfigPath)
}

func mcAppConfExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "multiclusterapplicationconfigurations",
	}
	return resourceExists(gvr, TestNamespace, appConfigName, kubeconfigPath)
}

func mcComponentExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "multiclustercomponents",
	}
	return resourceExists(gvr, TestNamespace, componentName, kubeconfigPath)
}

func componentWorkloadExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.GroupVersion.Group,
		Version:  oamv1alpha1.GroupVersion.Version,
		Resource: "verrazzanohelidonworkloads",
	}
	return resourceExists(gvr, TestNamespace, workloadName, kubeconfigPath)
}

func resourceExists(gvr schema.GroupVersionResource, ns string, name string, kubeconfigPath string) bool {
	config := pkg.GetKubeConfigGivenPath(kubeconfigPath)
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not create dynamic client: %v\n", err))
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

	return u != nil && err == nil
}

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(TestNamespace, expectedPodsHelloHelidon)
}
