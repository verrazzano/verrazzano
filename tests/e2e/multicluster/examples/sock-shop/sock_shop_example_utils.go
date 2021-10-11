// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"context"
	"fmt"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	multiclusterNamespace = "verrazzano-mc"
	appConfigName         = "sockshop-appconf"
)

var expectedCompsSockShop = []string{"carts-component", "catalog-component", "orders-component", "payment-component",
	"shipping-component", "users-component"}

var expectedWorkloads = []string{"carts-coh", "catalog-coh", "orders-coh", "payment-coh", "shipping-coh", "users-coh"}

var expectedPodsSockShop = []string{"carts-coh", "catalog-coh", "orders-coh", "payment-coh", "shipping-coh", "users-coh"}

// DeploySockShopProject deploys the sock-shop example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeploySockShopProject(kubeconfigPath string, sourceDir string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to create %s project resource: %v", sourceDir, err)
	}
	return nil
}

// DeploySockShopApp deploys the sock-shop example application to the cluster with the given kubeConfigPath
func DeploySockShopApp(kubeConfigPath string, sourceDir string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/sock-shop-comp.yaml", sourceDir), kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster %s component resources: %v", sourceDir, err)
	}
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/sock-shop-app.yaml", sourceDir), kubeConfigPath); err != nil {
		return fmt.Errorf("Failed to create multi-cluster %s application resource: %v", sourceDir, err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	// call both mcAppConfExists and mcComponentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := mcAppConfExists(kubeconfigPath, namespace)
	compExists := true

	// check each sock-shop component in expectedCompsSockShop
	for _, comp := range expectedCompsSockShop {
		compExists = componentExists(kubeconfigPath, namespace, comp) && compExists
	}

	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		return mcAppConfExists && compExists
	} else {
		// don't expect either
		return !mcAppConfExists && !compExists
	}
}

// VerifySockShopInCluster verifies that the sock-shop app resources are either present or absent
// depending on whether the app is placed in this cluster
func VerifySockShopInCluster(kubeConfigPath string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string) bool {
	projectExists := projectExists(kubeConfigPath, projectName)
	podsRunning := sockShopPodsRunning(kubeConfigPath, namespace)

	workloadExists := true
	// check each sock-shop workload in expectedWorkloads
	for _, workload := range expectedWorkloads {
		workloadExists = componentWorkloadExists(kubeConfigPath, namespace, workload) && workloadExists
	}

	if placedInThisCluster {
		return projectExists && podsRunning && workloadExists
	} else {
		if isAdminCluster {
			return projectExists && !podsRunning && !workloadExists
		} else {
			return !podsRunning && !projectExists && !workloadExists
		}
	}
}

// VerifySockShopDeleteOnAdminCluster verifies that the sock shop app resources have been deleted from the admin
// cluster after the application has been deleted
func VerifySockShopDeleteOnAdminCluster(kubeconfigPath string, placedInCluster bool, namespace string, projectName string) bool {
	mcResDeleted := VerifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	if !placedInCluster {
		return mcResDeleted
	}

	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted
}

// VerifySockShopDeleteOnManagedCluster verifies that the sock shop app resources have been deleted from the managed
// cluster after the application has been deleted
func VerifySockShopDeleteOnManagedCluster(kubeconfigPath string, namespace string, projectName string) bool {
	mcResDeleted := VerifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted

}

// VerifyAppDeleted verifies that the workload and pods are deleted on the specified cluster
func VerifyAppDeleted(kubeConfigPath string, namespace string) bool {

	workloadExists := true
	// check each sock-shop workload in expectedWorkloads
	for _, workload := range expectedWorkloads {
		workloadExists = componentWorkloadExists(kubeConfigPath, namespace, workload) && workloadExists
	}

	podsRunning := sockShopPodsRunning(kubeConfigPath, namespace)
	return !workloadExists && !podsRunning
}

// VerifyMCResourcesDeleted verifies that any resources created by the deployment are deleted on the specified cluster
func VerifyMCResourcesDeleted(kubeconfigPath string, namespace string, projectName string) bool {
	appConfExists := mcAppConfExists(kubeconfigPath, namespace)
	compExists := true

	// check each sock-shop component in expectedCompsSockShop
	for _, comp := range expectedCompsSockShop {
		compExists = componentExists(kubeconfigPath, namespace, comp) && compExists
	}
	projExists := projectExists(kubeconfigPath, projectName)
	return !appConfExists && !compExists && !projExists
}

// SockShopExists - returns true if the sock-shop namespace exists in the given cluster
func SockShopNamespaceExists(kubeconfigPath string, namespace string) bool {
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

// Check if individual component exists
func componentExists(kubeconfigPath string, namespace string, component string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: "components",
	}
	return resourceExists(gvr, namespace, component, kubeconfigPath)
}

func componentWorkloadExists(kubeconfigPath string, namespace string, workload string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.SchemeGroupVersion.Group,
		Version:  oamv1alpha1.SchemeGroupVersion.Version,
		Resource: "verrazzanocoherenceworkloads",
	}
	return resourceExists(gvr, namespace, workload, kubeconfigPath)
}

func resourceExists(gvr schema.GroupVersionResource, ns string, name string, kubeconfigPath string) bool {
	config, err := pkg.GetKubeConfigGivenPath(kubeconfigPath)
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

func sockShopPodsRunning(kubeconfigPath string, namespace string) bool {
	return pkg.PodsRunningInCluster(namespace, expectedPodsSockShop, kubeconfigPath)
}
