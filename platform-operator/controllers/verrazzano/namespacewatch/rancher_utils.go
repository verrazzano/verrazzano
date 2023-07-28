// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	RancherManagedNamespaceLabelKey  = "management.cattle.io/system-namespace"
	RancherProjectIDLabelKey         = "field.cattle.io/projectId"
	APIGroupRancherManagement        = "management.cattle.io"
	APIGroupVersionRancherManagement = "v3"
)

// getRancherProjectList returns the list of Rancher projects
func getRancherProjectList(dynClient dynamic.Interface) (*unstructured.UnstructuredList, error) {
	var rancherProjectList *unstructured.UnstructuredList
	gvr := GetRancherMgmtAPIGVRForResource("projects")
	rancherProjectList, err := dynClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get %s/%s/%s: %v", gvr.Resource, gvr.Group, gvr.Version, err)
	}
	return rancherProjectList, nil
}

// getRancherSystemProjectID returns the ID of Rancher system project
func getRancherSystemProjectID(dynClient dynamic.Interface) string {
	var projectID string
	rancherProjectList, _ := getRancherProjectList(dynClient)
	for _, rancherProject := range rancherProjectList.Items {
		projectName := rancherProject.UnstructuredContent()["spec"].(map[string]interface{})["displayName"].(string)
		if projectName == "System" {
			projectID = rancherProject.UnstructuredContent()["metadata"].(map[string]interface{})["name"].(string)
		}
	}
	return projectID
}

// GetRancherMgmtAPIGVRForResource returns a management.cattle.io/v3 GroupVersionResource structure for specified kind
func GetRancherMgmtAPIGVRForResource(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    APIGroupRancherManagement,
		Version:  APIGroupVersionRancherManagement,
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