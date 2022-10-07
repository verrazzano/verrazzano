// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"time"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	multiclusterNamespace = "verrazzano-mc"
	appConfigName         = "todo-appconf"
	shortWaitTimeout      = 10 * time.Minute
	shortPollingInterval  = 10 * time.Second
)

var (
	expectedCompsTodoList = []string{
		"todo-domain",
		"todo-jdbc-config",
		"mysql-initdb-config",
		"todo-mysql-service",
		"todo-mysql-deployment"}
	expectedPodsTodoList = []string{
		"mysql",
		"tododomain-adminserver"}
)

// DeployTodoListProject deploys the todo-list example's VerrazzanoProject to the cluster with the given kubeConfigPath
func DeployTodoListProject(kubeconfigPath string, sourceDir string) error {
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create %s project resource: %v", sourceDir, err)
	}
	return nil
}

// TodoListNamespaceExists returns true if the todo-list namespace exists in the given cluster
func TodoListNamespaceExists(kubeconfigPath string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeconfigPath)
	return err == nil
}

// DeployTodoListApp deploys the todo-list example application to the cluster with the given kubeConfigPath
func DeployTodoListApp(kubeconfigPath string, sourceDir string) error {
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/todo-list-components.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s component resources: %v", sourceDir, err)
	}
	if err := resource.CreateOrUpdateResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-todo-list-application.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to create multi-cluster %s application resource: %v", sourceDir, err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, namespace string) bool {
	// call both appConfExists and componentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := appConfExists(kubeconfigPath, namespace)

	compExists := true
	// check each todo-list component in expectedCompsTodoList
	for _, comp := range expectedCompsTodoList {
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

// VerifyTodoListInCluster verifies that the todo-list app resources are either present or absent
// depending on whether the app is placed in this cluster
func VerifyTodoListInCluster(kubeconfigPath string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string) (bool, error) {
	projectExists := projectExists(kubeconfigPath, projectName)
	podsRunning, err := todoListPodsRunning(kubeconfigPath, namespace)
	if err != nil {
		return false, err
	}

	if placedInThisCluster {
		return projectExists && podsRunning, nil
	} else {
		if isAdminCluster {
			return projectExists && !podsRunning, nil
		} else {
			return !podsRunning && !projectExists, nil
		}
	}
}

// projectExists Check if todo-list project exists
func projectExists(kubeconfigPath string, projectName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName, kubeconfigPath)
}

// todoListPodsRunning Check if expected pods are running on a given cluster
func todoListPodsRunning(kubeconfigPath string, namespace string) (bool, error) {
	result, err := pkg.PodsRunningInCluster(namespace, expectedPodsTodoList, kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		return false, err
	}
	return result, nil
}

// todoListPodDeleted Check if expected pods are running on a given cluster
func todoListPodDeleted(kubeconfigPath string, namespace string, pod string) bool {
	deletedPod := []string{pod}
	result, _ := pkg.PodsRunningInCluster(namespace, deletedPod, kubeconfigPath)
	return !result
}

// VerifyTodoListDeleteOnAdminCluster verifies that the todo-list app resources have been deleted from the admin
// cluster after the application has been deleted
func VerifyTodoListDeleteOnAdminCluster(kubeconfigPath string, placedInCluster bool, namespace string, projectName string) bool {
	mcResDeleted := VerifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	if !placedInCluster {
		return mcResDeleted
	}
	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted
}

// VerifyTodoListDeleteOnManagedCluster verifies that the todo-list app resources have been deleted from the managed
// cluster after the application has been deleted
func VerifyTodoListDeleteOnManagedCluster(kubeconfigPath string, namespace string, projectName string) bool {
	mcResDeleted := VerifyMCResourcesDeleted(kubeconfigPath, namespace, projectName)
	appDeleted := VerifyAppDeleted(kubeconfigPath, namespace)

	return mcResDeleted && appDeleted

}

// VerifyAppDeleted verifies that the workload and pods are deleted on the specified cluster
func VerifyAppDeleted(kubeconfigPath string, namespace string) bool {

	podsDeleted := true
	// check that each todo-list pod is deleted
	for _, pod := range expectedPodsTodoList {
		podsDeleted = todoListPodDeleted(namespace, pod, kubeconfigPath) && podsDeleted
	}

	return podsDeleted
}

// VerifyMCResourcesDeleted verifies that any resources created by the deployment are deleted on the specified cluster
func VerifyMCResourcesDeleted(kubeconfigPath string, namespace string, projectName string) bool {
	appConfExists := appConfExists(kubeconfigPath, namespace)
	projExists := projectExists(kubeconfigPath, projectName)

	compExists := true
	// check each todo-list component in expectedCompsTodoList
	for _, comp := range expectedCompsTodoList {
		compExists = componentExists(kubeconfigPath, namespace, comp) && compExists
	}

	return !appConfExists && !compExists && !projExists
}
