// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multicluster

import (
	"context"
	"fmt"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

const (
	comps        = "components"
	mcAppConfigs = "multiclusterapplicationconfigurations"
	mcNamespace  = "verrazzano-mc"
	projects     = "verrazzanoprojects"
)

// DeployVerrazzanoProject deploys the VerrazzanoProject to the cluster with the given kubeConfig
func DeployVerrazzanoProject(projectConfiguration, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(projectConfiguration, kubeConfig); err != nil {
		return fmt.Errorf("failed to create project resource: %v", err)
	}
	return nil
}

// TestNamespaceExists returns true if the test namespace exists in the given cluster
func TestNamespaceExists(kubeConfig string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeConfig)
	return err == nil
}

// DeployCompResource deploys the OAM Component resource to the cluster with the given kubeConfig
func DeployCompResource(compConfiguration, testNamespace, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(compConfiguration, kubeConfig, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster component resources: %v", err)
	}
	return nil
}

// DeployAppResource deploys the OAM Application resource to the cluster with the given kubeConfig
func DeployAppResource(appConfiguration, testNamespace, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(appConfiguration, kubeConfig, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster application resource: %v", err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeConfig string, isAdminCluster bool, placedInThisCluster bool, namespace string, appConfigName string, expectedComps []string) bool {
	// call both appConfExists and componentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := appConfExists(kubeConfig, namespace, appConfigName)

	compExists := true
	// check each component in expectedComps
	for _, comp := range expectedComps {
		compExists = componentExists(kubeConfig, namespace, comp) && compExists
	}

	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		return mcAppConfExists && compExists
	}
	// don't expect either
	return !mcAppConfExists && !compExists
}

// VerifyAppResourcesInCluster verifies that the app resources are either present or absent
// depending on whether the app is placed in this cluster
func VerifyAppResourcesInCluster(kubeConfig string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string, appPods []string) (bool, error) {
	projectExists := projectExists(kubeConfig, projectName)
	podsRunning, err := checkPodsRunning(kubeConfig, namespace, appPods)
	if err != nil {
		return false, err
	}

	if placedInThisCluster {
		return projectExists && podsRunning, nil
	}
	if isAdminCluster {
		return projectExists && !podsRunning, nil
	}
	return !podsRunning && !projectExists, nil
}

// VerifyDeleteOnAdminCluster verifies that the app resources have been deleted from the admin
// cluster after the application has been deleted
func VerifyDeleteOnAdminCluster(kubeConfig string, placedInCluster bool, namespace string, projectName string, appConfigName string, appPods []string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeConfig, namespace, projectName, appConfigName, appPods)
	if !placedInCluster {
		return mcResDeleted
	}
	appDeleted := verifyAppDeleted(kubeConfig, namespace, appPods)
	return mcResDeleted && appDeleted
}

// VerifyDeleteOnManagedCluster verifies that the app resources have been deleted from the managed
// cluster after the application has been deleted
func VerifyDeleteOnManagedCluster(kubeConfig string, namespace string, projectName string, appConfigName string, appPods []string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeConfig, namespace, projectName, appConfigName, appPods)
	appDeleted := verifyAppDeleted(kubeConfig, namespace, appPods)

	return mcResDeleted && appDeleted
}

// appConfExists Check if app config exists
func appConfExists(kubeConfig string, namespace string, appConfigName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: mcAppConfigs,
	}
	return resourceExists(gvr, namespace, appConfigName, kubeConfig)
}

// resourceExists Check if given resource exists
func resourceExists(gvr schema.GroupVersionResource, ns string, name string, kubeConfig string) bool {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeConfig)
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

// componentExists Check if individual component exists
func componentExists(kubeConfig string, namespace string, component string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: comps,
	}
	return resourceExists(gvr, namespace, component, kubeConfig)
}

// projectExists Check if project with name projectName exists
func projectExists(kubeConfig string, projectName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: projects,
	}
	return resourceExists(gvr, mcNamespace, projectName, kubeConfig)
}

// checkPodsRunning Check if expected pods are running on a given cluster
func checkPodsRunning(kubeConfig string, namespace string, appPods []string) (bool, error) {
	result, err := pkg.PodsRunningInCluster(namespace, appPods, kubeConfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		return false, err
	}
	return result, nil
}

// verifyAppDeleted verifies that the workload and pods are deleted on the specified cluster
func verifyAppDeleted(kubeConfig string, namespace string, appPods []string) bool {
	podsDeleted := true
	// check that each pod is deleted
	for _, pod := range appPods {
		podsDeleted = checkPodDeleted(namespace, pod, kubeConfig) && podsDeleted
	}
	return podsDeleted
}

// checkPodDeleted Check if expected pods are running on a given cluster
func checkPodDeleted(kubeConfig string, namespace string, pod string) bool {
	deletedPod := []string{pod}
	result, _ := pkg.PodsRunningInCluster(namespace, deletedPod, kubeConfig)
	return !result
}

// verifyMCResourcesDeleted verifies that any resources created by the deployment are deleted on the specified cluster
func verifyMCResourcesDeleted(kubeConfig string, namespace string, projectName string, appConfigName string, appPods []string) bool {
	appConfExists := appConfExists(kubeConfig, namespace, appConfigName)
	projExists := projectExists(kubeConfig, projectName)

	compExists := true
	// check each component in appPods
	for _, comp := range appPods {
		compExists = componentExists(kubeConfig, namespace, comp) && compExists
	}

	return !appConfExists && !compExists && !projExists
}
