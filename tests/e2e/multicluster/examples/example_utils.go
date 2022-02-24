// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package examples

import (
	"context"
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	TestNamespace         = "hello-helidon" // currently only used for placement tests
	multiclusterNamespace = "verrazzano-mc"
	projectName           = "hello-helidon"
	appConfigName         = "hello-helidon-appconf"
	componentName         = "hello-helidon-component"
	workloadName          = "hello-helidon-workload"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}

// DeployHelloHelidonProject deploys the hello-helidon example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeployHelloHelidonProject(kubeconfigPath string, sourceDir string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create %s project resource: %v", sourceDir, err)
	}
	return nil
}

// DeployHelloHelidonApp deploys the hello-helidon example application to the cluster with the given kubeConfigPath
func DeployHelloHelidonApp(kubeConfigPath string, sourceDir string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/hello-helidon-comp.yaml", sourceDir), kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s component resources: %v", sourceDir, err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-hello-helidon-app.yaml", sourceDir), kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s application resource: %v", sourceDir, err)
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
	mcAppGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.MultiClusterAppConfigResource)
	vpGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.VerrazzanoProjectResource)

	if err := pkg.PatchResourceFromFileInCluster(mcAppGvr, TestNamespace, appConfigName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon application resource: %v", err)
	}
	if err := pkg.PatchResourceFromFileInCluster(vpGvr, multiclusterNamespace, projectName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create VerrazzanoProject resource: %v", err)
	}
	return nil
}

// ChangePlacementV100 patches the hello-helidon example with the given patch file
// and uses the given kubeConfigPath as the cluster in which to do the patch
// v1.0.0 variant of this function - requires edit to placement in mcComp resources
func ChangePlacementV100(kubeConfigPath string, patchFile string, namespace string, projName string) error {
	mcCompGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.MultiClusterComponentResource)
	mcAppGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.MultiClusterAppConfigResource)
	vpGvr := clustersv1alpha1.SchemeGroupVersion.WithResource(clustersv1alpha1.VerrazzanoProjectResource)

	if err := pkg.PatchResourceFromFileInCluster(mcCompGvr, namespace, componentName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon component resource: %v", err)
	}
	if err := pkg.PatchResourceFromFileInCluster(mcAppGvr, namespace, appConfigName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon application resource: %v", err)
	}
	if err := pkg.PatchResourceFromFileInCluster(vpGvr, multiclusterNamespace, projName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create VerrazzanoProject resource: %v", err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	mcAppConfExists := mcAppConfExists(kubeconfigPath, namespace)

	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		return mcAppConfExists
	} else {
		// don't expect
		return !mcAppConfExists
	}
}

// VerifyMCResourcesV100 verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
// v1.0.0 variant of this function - both mcApp and mcComp are required
func VerifyMCResourcesV100(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	// call both mcAppConfExists and mcComponentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := mcAppConfExists(kubeconfigPath, namespace)
	mcCompExists := mcComponentExists(kubeconfigPath, namespace)

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
func VerifyHelloHelidonInCluster(kubeConfigPath string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string) bool {
	projectExists := projectExists(kubeConfigPath, projectName)
	workloadExists := componentWorkloadExists(kubeConfigPath, namespace)
	podsRunning := helloHelidonPodsRunning(kubeConfigPath, namespace)

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

func VerifyHelloHelidonDeletedAdminCluster(kubeconfigPath string, placedInCluster bool, namespace string, projectName string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	if !placedInCluster {
		return mcResDeleted
	}

	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted
}

func VerifyHelloHelidonDeletedInManagedCluster(kubeconfigPath string, namespace string, projectName string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted

}

// VerifyAppDeleted - verifies that the workload and pods are deleted on the the specified cluster
func VerifyAppDeleted(kubeconfigPath string, namespace string) bool {
	workloadExists := componentWorkloadExists(kubeconfigPath, namespace)
	podsRunning := helloHelidonPodsRunning(kubeconfigPath, namespace)
	return !workloadExists && !podsRunning
}

func verifyMCResourcesDeleted(kubeconfigPath string, namespace string, projectName string) bool {
	appConfExists := mcAppConfExists(kubeconfigPath, namespace)
	mcCompExists := mcComponentExists(kubeconfigPath, namespace)
	projExists := projectExists(kubeconfigPath, projectName)
	compExists := componentExists(kubeconfigPath, namespace)
	return !appConfExists && !compExists && !projExists && !mcCompExists
}

// HelidonNamespaceExists - returns true if the hello-helidon namespace exists in the given cluster
func HelidonNamespaceExists(kubeconfigPath string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeconfigPath)
	return err == nil
}

func projectExists(kubeconfigPath string, projectName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName, kubeconfigPath)
}

func mcAppConfExists(kubeconfigPath string, namespace string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "multiclusterapplicationconfigurations",
	}
	return resourceExists(gvr, namespace, appConfigName, kubeconfigPath)
}

func mcComponentExists(kubeconfigPath string, namespace string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "multiclustercomponents",
	}
	return resourceExists(gvr, namespace, componentName, kubeconfigPath)
}

func componentExists(kubeconfigPath string, namespace string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.SchemeGroupVersion.Group,
		Version:  oamv1alpha1.SchemeGroupVersion.Version,
		Resource: "components",
	}
	return resourceExists(gvr, namespace, componentName, kubeconfigPath)
}

func componentWorkloadExists(kubeconfigPath string, namespace string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.SchemeGroupVersion.Group,
		Version:  oamv1alpha1.SchemeGroupVersion.Version,
		Resource: "verrazzanohelidonworkloads",
	}
	return resourceExists(gvr, namespace, workloadName, kubeconfigPath)
}

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

func helloHelidonPodsRunning(kubeconfigPath string, namespace string) bool {
	// TODO: Go through each of the test cases calling this and decide whether to fail the test or test suite
	result, err := pkg.PodsRunningInCluster(namespace, expectedPodsHelloHelidon, kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}
