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
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	TestNamespace   = "hello-helidon"
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute

	multiclusterNamespace = "verrazzano-mc"
	projectName           = "hello-helidon"
	appConfigName         = "hello-helidon-appconf"
	componentName         = "hello-helidon-component"
	workloadName          = "hello-helidon-workload"
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

// ChangePlacementToAdminCluster patches the hello-helidon example to be placed in the admin cluster
// and uses the given kubeConfigPath as the cluster in which to do the patch
func ChangePlacementToAdminCluster(kubeconfigPath string) error {
	return changePlacement(kubeconfigPath, "examples/multicluster/hello-helidon/patch-change-placement-to-admin.yaml")
}

// ChangePlacementToManagedCluster patches the hello-helidon example to be placed in the managed cluster
// and uses the given kubeConfigPath as the cluster in which to do the patch
func ChangePlacementToManagedCluster(kubeconfigPath string) error {
	return changePlacement(kubeconfigPath, "examples/multicluster/hello-helidon/patch-return-placement-to-managed1.yaml")
}

// changePlacement patches the hello-helidon example with the given patch file
// and uses the given kubeConfigPath as the cluster in which to do the patch
func changePlacement(kubeConfigPath string, patchFile string) error {
	mcCompGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.MultiClusterComponentResource)
	mcAppGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.MultiClusterAppConfigResource)
	vpGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.VerrazzanoProjectResource)

	if err := pkg.PatchResourceFromFileInCluster(mcCompGvr, TestNamespace, componentName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to change placement of multicluster hello-helidon component resource: %v", err)
	}
	if err := pkg.PatchResourceFromFileInCluster(mcAppGvr, TestNamespace, appConfigName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to change placement of multicluster hello-helidon application resource: %v", err)
	}
	if err := pkg.PatchResourceFromFileInCluster(vpGvr, multiclusterNamespace, projectName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create VerrazzanoProject resource: %v", err)
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
func VerifyHelloHelidonInCluster(kubeConfigPath string, isAdminCluster bool, placedInThisCluster bool) bool {
	projectExists := projectExists(kubeConfigPath)
	workloadExists := componentWorkloadExists(kubeConfigPath)
	podsRunning := helloHelidonPodsRunning(kubeConfigPath)

	if placedInThisCluster {
		return projectExists && workloadExists && podsRunning
	} else {
		if isAdminCluster {
			return projectExists && !workloadExists && !podsRunning
		} else {
			return !workloadExists && !podsRunning && !projectExists
		}
	}
}

func VerifyHelloHelidonDeletedAdminCluster(kubeconfigPath string, placedInCluster bool) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath)
	if !placedInCluster {
		return mcResDeleted
	}

	appDeleted := VerifyAppDeleted(kubeconfigPath)

	return mcResDeleted && appDeleted
}

func VerifyHelloHelidonDeletedInManagedCluster(kubeconfigPath string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath)
	appDeleted := VerifyAppDeleted(kubeconfigPath)

	return mcResDeleted && appDeleted

}

// VerifyAppDeleted - verifies that the workload and pods are deleted on the the specified cluster
func VerifyAppDeleted(kubeconfigPath string) bool {
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
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName, kubeconfigPath)
}

func mcAppConfExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "multiclusterapplicationconfigurations",
	}
	return resourceExists(gvr, TestNamespace, appConfigName, kubeconfigPath)
}

func mcComponentExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "multiclustercomponents",
	}
	return resourceExists(gvr, TestNamespace, componentName, kubeconfigPath)
}

func componentWorkloadExists(kubeconfigPath string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.SchemeGroupVersion.Group,
		Version:  oamv1alpha1.SchemeGroupVersion.Version,
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
		if errors.IsNotFound(err) {
			return false
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Could not retrieve resource %s: %v\n", gvr.String(), err))
		return false
	}
	return u != nil
}

func helloHelidonPodsRunning(kubeconfigPath string) bool {
	return pkg.PodsRunningInCluster(TestNamespace, expectedPodsHelloHelidon, kubeconfigPath)
}
