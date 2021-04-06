// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package examples

import (
	"context"
	"fmt"
	"time"

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

// DeployHelloHelidonApp deploys the hello-helidon example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeployHelloHelidonProject(kubeconfigPath string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to create hello-helidon project resource: %v", err)
	}
	return nil
}

// DeployHelloHelidonApp deploys the hello-helidon example application to the cluster with the given kubeConfigPath
func DeployHelloHelidonApp(kubeConfigPath string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml", kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster hello-helidon component resources: %v", err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml", kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster hello-helidon application resource: %v", err)
	}
	return nil
}

// DeployChangePlacement deploys the change-placement example to the cluster with the given kubeConfigPath
func DeployChangePlacement(kubeConfigPath string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-comp.yaml", kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster hello-helidon component resources: %v", err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster("examples/multicluster/change-placement/mc-hello-helidon-app.yaml", kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster hello-helidon application resource: %v", err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool) bool {
	// call both mcAppConfExists and mcComponentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := mcAppConfExists(kubeconfigPath)
	mcCompExists := mcComponentExists(kubeconfigPath)

	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		return mcAppConfExists && mcCompExists
	} else {
		// don't expect either
		return !mcAppConfExists && !mcCompExists
	}
}

// VerifyHelloHelidonInCluster verifies that the hello helidon app resources are either present or absent
// depending on whether the app is placed in this cluster
func VerifyHelloHelidonInCluster(kubeConfigPath string, placedInThisCluster bool) bool {
	projectExists := projectExists(kubeConfigPath)
	workloadExists := componentWorkloadExists(kubeConfigPath)
	podsRunning := helloHelidonPodsRunning(kubeConfigPath)

	if !placedInThisCluster {
		return projectExists && !workloadExists && !podsRunning
	} else {
		return projectExists && workloadExists && podsRunning
	}
}

func VerifyHelloHelidonDeletedAdminCluster(kubeconfigPath string, placedInCluster bool) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath)
	if !placedInCluster {
		return mcResDeleted
	}

	appDeleted := verifyAppDeleted(kubeconfigPath)

	return mcResDeleted && appDeleted
}

func VerifyHelloHelidonDeletedInManagedCluster(kubeconfigPath string) bool {
	return !projectExists(kubeconfigPath)

	// NOTE: These tests are disabled pending a fix for VZ-2454 - once fixed, the project check can be removed since
	// it is part of verifyMCResourcesDeleted. It is here because it is the only thing that can be verified on a
	// managed cluster due to the bug

	/*
		mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath)
		appDeleted := verifyAppDeleted(kubeconfigPath)

		return mcResDeleted && appDeleted

	*/
}

func verifyAppDeleted(kubeconfigPath string) bool {
	workloadExists := componentWorkloadExists(kubeconfigPath)
	podsRunning := helloHelidonPodsRunning(kubeconfigPath)
	return !workloadExists && !podsRunning
}

func verifyMCResourcesDeleted(kubeconfigPath string) bool {
	appConfExists := mcAppConfExists(kubeconfigPath)
	compExists := mcComponentExists(kubeconfigPath)
	projExists := projectExists(kubeconfigPath)
	return !appConfExists && !compExists && !projExists
}

// HelidonNamespaceExists - returns true if the hello-helidon namespace exists in the given cluster
func HelidonNamespaceExists(kubeconfigPath string) bool {
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
		pkg.Log(pkg.Error, fmt.Sprintf("Could not create dynamic client: %v\n", err))
		return false
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not retrieve resource %s: %v\n", gvr.String(), err))
		return false
	}
	return u != nil
}

func helloHelidonPodsRunning(kubeconfigPath string) bool {
	return pkg.PodsRunningInCluster(TestNamespace, expectedPodsHelloHelidon, kubeconfigPath)
}
