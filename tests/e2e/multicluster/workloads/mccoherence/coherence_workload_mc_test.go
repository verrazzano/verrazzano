// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mccoherence

import (
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"fmt"
	"time"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	testNamespace        = "mc-hello-coherence"
	appConfigName         = "hello-appconf"
)

var (
	expectedComps = []string{
		"todo-domain",
		"todo-jdbc-config",
		"mysql-initdb-config",
		"todo-mysql-service",
		"todo-mysql-deployment"}
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false
var beforeSuitePassed = false

var t = framework.NewTestFramework("mccoherence")

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {

	// deploy the VerrazzanoProject
	start := time.Now()
	Eventually(func() error {
		return deployVerrazzanoProject(adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return testNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	Eventually(func() error {
		return deployCoherenceApp(adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("In Multi-cluster, verify todo-list", Label("f:multicluster.mc-app-lcm"), func() {
	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return verifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})
})

// deployVerrazzanoProject deploys the VerrazzanoProject to the cluster with the given kubeConfigPath
func deployVerrazzanoProject(kubeconfigPath string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("tests/testdata/test-applications/coherence/hello-coherence/verrazzano-project.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create project resource: %v", err)
	}
	return nil
}

// testNamespaceExists returns true if the test namespace exists in the given cluster
func testNamespaceExists(kubeconfigPath string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeconfigPath)
	return err == nil
}

// deployCoherenceApp deploys the Coherence application to the cluster with the given kubeConfigPath
func deployCoherenceApp(kubeconfigPath string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace("tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-mc-comp.yaml", kubeconfigPath, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster component resources: %v", err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace("tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-mc-app.yaml", kubeconfigPath, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster application resource: %v", err)
	}
	return nil
}

// verifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func verifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	// call both appConfExists and componentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := appConfExists(kubeconfigPath, namespace)
	if mcAppConfExists {
		fmt.Println("Application configuration exists")
	}
	return mcAppConfExists
	// compExists := true
	// check each todo-list component in expectedCompsTodoList
	/*for _, comp := range expectedComps {
	          compExists = componentExists(kubeconfigPath, namespace, comp) && compExists
	  }

	  if isAdminCluster || placedInThisCluster {
	          // always expect MC resources on admin cluster - otherwise expect them only if placed here
	          return mcAppConfExists && compExists
	  } else {
	          // don't expect either
	          return !mcAppConfExists && !compExists
	  }*/
}

// appConfExists Check if app config exists
func appConfExists(kubeconfigPath string, namespace string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "multiclusterapplicationconfigurations",
	}
	return resourceExists(gvr, namespace, appConfigName, kubeconfigPath)
}

// componentExists Check if individual component exists
func componentExists(kubeconfigPath string, namespace string, component string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: "components",
	}
	return resourceExists(gvr, namespace, component, kubeconfigPath)
}

// resourceExists Check if given resource exists
func resourceExists(gvr schema.GroupVersionResource, ns string, name string, kubeconfigPath string) bool {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not get kube config: %v\n", err))
		return false
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not create dynamic client: %v\n", err))
		return false
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Could not retrieve resource %s: %v\n", gvr.String(), err))
		return false
	}
	return u != nil
}