// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	RancherProjectIDLabelKey = "field.cattle.io/projectId"
)

// getRancherProjectList returns the list of Rancher projects
func getRancherProjectList(dynClient dynamic.Interface) (*unstructured.UnstructuredList, error) {
	var rancherProjectList *unstructured.UnstructuredList
	gvr := GetRancherMgmtAPIGVRForResource("projects")
	rancherProjectList, err := dynClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s/%s/%s: %v", gvr.Resource, gvr.Group, gvr.Version, err)
	}
	return rancherProjectList, nil
}

// getRancherSystemProjectID returns the ID of Rancher system project
func (nw *NamespacesWatcher) getRancherSystemProjectID() (string, string, error) {

	isRancherReady, err := nw.IsRancherReady()
	if err != nil {
		return "", "", err
	}
	if !isRancherReady {
		nw.log.Debugf("rancher is not enabled or ready")
		return "", "", nil
	}
	dynClient, err := getDynamicClient()
	if err != nil {
		return "", "", err
	}
	rancherProjectList, err := getRancherProjectList(dynClient)
	if err != nil {
		return "", "", err
	}
	var projectID string
	for _, rancherProject := range rancherProjectList.Items {
		projectName := rancherProject.UnstructuredContent()["spec"].(map[string]interface{})["displayName"].(string)
		clusterName := rancherProject.UnstructuredContent()["spec"].(map[string]interface{})["clusterName"].(string)
		if projectName == "System" {
			projectID = rancherProject.UnstructuredContent()["metadata"].(map[string]interface{})["name"].(string)
			return clusterName, projectID, nil
		}
	}
	return "", "", nil
}

// GetRancherMgmtAPIGVRForResource returns a management.cattle.io/v3 GroupVersionResource structure for specified kind
func GetRancherMgmtAPIGVRForResource(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    common.APIGroupRancherManagement,
		Version:  common.APIGroupVersionRancherManagement,
		Resource: resource,
	}
}

// GetDynamicClient returns a dynamic client needed to access Unstructured data
func getDynamicClient() (dynamic.Interface, error) {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}

func (nw *NamespacesWatcher) IsRancherReady() (bool, error) {
	vz, err := getVerrazzanoResource(nw.client)
	if err != nil {
		return false, fmt.Errorf("failed to get Verrazzano resource: %v", err)
	}

	logger, err := newLogger(vz)
	if err != nil {
		return false, fmt.Errorf("failed to get Verrazzano resource logger: %v", err)
	}
	ctx, err := spi.NewContext(logger, nw.client, vz, nil, true)
	if err != nil {
		return false, fmt.Errorf("error creating a component context %v", err)
	}
	_, rancherComponent := registry.FindComponent(common.RancherName)
	isEnabled := rancherComponent.IsEnabled(ctx.EffectiveCR())
	return isEnabled && rancherComponent.IsReady(ctx), nil
}
