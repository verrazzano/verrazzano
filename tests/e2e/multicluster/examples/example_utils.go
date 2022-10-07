// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package examples

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	appopconst "github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	helloHelidon          = "hello-helidon"
	TestNamespace         = "hello-helidon" // currently only used for placement tests
	multiclusterNamespace = "verrazzano-mc"
	projectName           = helloHelidon
	appConfigName         = helloHelidon
	componentName         = "hello-helidon-component"
	workloadName          = "hello-helidon-workload"
	oamGroup              = "core.oam.dev"
	oamVersion            = "v1alpha2"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
var compGvr = schema.GroupVersionResource{
	Group:    oamGroup,
	Version:  oamVersion,
	Resource: "components",
}
var appConfGvr = schema.GroupVersionResource{
	Group:    oamGroup,
	Version:  oamVersion,
	Resource: "applicationconfigurations",
}

var projectGvr = schema.GroupVersionResource{
	Group:    clustersv1alpha1.SchemeGroupVersion.Group,
	Version:  clustersv1alpha1.SchemeGroupVersion.Version,
	Resource: "verrazzanoprojects",
}

// DeployHelloHelidonProject deploys the hello-helidon example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeployHelloHelidonProject(kubeconfigPath string, sourceDir string) error {
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create %s project resource: %v", sourceDir, err)
	}
	return nil
}

// DeployHelloHelidonApp deploys the hello-helidon example application to the cluster with the given kubeConfigPath
func DeployHelloHelidonApp(kubeConfigPath string, sourceDir string) error {
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/hello-helidon-comp.yaml", sourceDir), kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s component resources: %v", sourceDir, err)
	}
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-hello-helidon-app.yaml", sourceDir), kubeConfigPath); err != nil {
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

	if err := resource.PatchResourceFromFileInCluster(mcAppGvr, TestNamespace, appConfigName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon application resource: %v", err)
	}
	if err := resource.PatchResourceFromFileInCluster(vpGvr, multiclusterNamespace, projectName, patchFile, kubeConfigPath); err != nil {
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

	if err := resource.PatchResourceFromFileInCluster(mcCompGvr, namespace, componentName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon component resource: %v", err)
	}
	if err := resource.PatchResourceFromFileInCluster(mcAppGvr, namespace, appConfigName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to change placement of multicluster hello-helidon application resource: %v", err)
	}
	if err := resource.PatchResourceFromFileInCluster(vpGvr, multiclusterNamespace, projName, patchFile, kubeConfigPath); err != nil {
		return fmt.Errorf("failed to create VerrazzanoProject resource: %v", err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	mcAppConfExists := mcAppConfExists(kubeconfigPath, namespace)
	vzManagedLabelExists := verrazzanoManagedLabelExists(kubeconfigPath, namespace)
	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		if placedInThisCluster {
			// the verrazzano-managed label will exist on unwrapped resources in the cluster where
			// app is placed
			return mcAppConfExists && vzManagedLabelExists
		} else {
			return mcAppConfExists && !vzManagedLabelExists
		}

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
func VerifyHelloHelidonInCluster(kubeConfigPath string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string) (bool, error) {
	projectExists := projectExists(kubeConfigPath, projectName)
	workloadExists := componentWorkloadExists(kubeConfigPath, namespace)
	podsRunning, err := helloHelidonPodsRunning(kubeConfigPath, namespace)
	if err != nil {
		return false, err
	}

	if placedInThisCluster {
		return projectExists && workloadExists && podsRunning, nil
	} else {
		if isAdminCluster {
			return projectExists && !workloadExists && !podsRunning, nil
		} else {
			return !workloadExists && !podsRunning && !projectExists, nil
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
	podsRunning, _ := helloHelidonPodsRunning(kubeconfigPath, namespace)
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
	return resourceExists(projectGvr, multiclusterNamespace, projectName, kubeconfigPath)
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

func verrazzanoManagedLabelExists(kubeconfigPath string, namespace string) bool {
	appconf, err := getResource(appConfGvr, namespace, appConfigName, kubeconfigPath)
	if err != nil {
		return false
	}
	comp, err := getResource(compGvr, namespace, componentName, kubeconfigPath)
	if err != nil {
		return false
	}
	appLabels := appconf.GetLabels()
	compLabels := comp.GetLabels()
	return appLabels[vzconst.VerrazzanoManagedLabelKey] == appopconst.LabelVerrazzanoManagedDefault &&
		compLabels[vzconst.VerrazzanoManagedLabelKey] == appopconst.LabelVerrazzanoManagedDefault
}

func componentExists(kubeconfigPath string, namespace string) bool {
	return resourceExists(compGvr, namespace, componentName, kubeconfigPath)
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
	u, err := getResource(gvr, ns, name, kubeconfigPath)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Could not retrieve resource %s: %v\n", gvr.String(), err))
		return false
	}
	return u != nil
}

func getResource(gvr schema.GroupVersionResource, ns string, name string, kubeconfigPath string) (*unstructured.Unstructured, error) {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not get kube config: %v\n", err))
		return nil, err
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not create dynamic client: %v\n", err))
		return nil, err
	}

	return client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

}
func helloHelidonPodsRunning(kubeconfigPath string, namespace string) (bool, error) {
	result, err := pkg.PodsRunningInCluster(namespace, expectedPodsHelloHelidon, kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		return false, err
	}
	return result, nil
}
