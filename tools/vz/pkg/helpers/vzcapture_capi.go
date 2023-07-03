// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	v1Alpha1       = "v1alpha1"
	v1Beta1        = "v1beta1"
	addonsGroup    = "addons.cluster.x-k8s.io"
	bootstrapGroup = "bootstrap.cluster.x-k8s.io"
	clusterGroup   = "cluster.x-k8s.io"
)

type capiResource struct {
	GVR  schema.GroupVersionResource
	Kind string
}

var capiResources = []capiResource{
	{GVR: schema.GroupVersionResource{Group: addonsGroup, Version: v1Beta1, Resource: "clusterresourcesetbindings"}, Kind: "ClusterResourceSetBindings"},
	{GVR: schema.GroupVersionResource{Group: addonsGroup, Version: v1Beta1, Resource: "clusterresourcesets"}, Kind: "ClusterResourceSets"},
	{GVR: schema.GroupVersionResource{Group: bootstrapGroup, Version: v1Alpha1, Resource: "ocneconfigs"}, Kind: "OCNEConfigs"},
	{GVR: schema.GroupVersionResource{Group: bootstrapGroup, Version: v1Alpha1, Resource: "ocneconfigtemplates"}, Kind: "OCNEConfigTemplates"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "clusterclasses"}, Kind: "ClusterClasses"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "clusters"}, Kind: "Clusters"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinedeployments"}, Kind: "MachineDeployments"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinehealthchecks"}, Kind: "MachineHealthChecks"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinepools"}, Kind: "MachinePools"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machines"}, Kind: "Machines"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinesets"}, Kind: "MachineSets"},
}

// captureCapiResources captures resources related to ClusterAPI
func captureCapiResources(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {

	for _, resource := range capiResources {
		if err := captureResource(dynamicClient, resource.GVR, resource.Kind, namespace, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

func captureResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, kind string, namespace, captureDir string, vzHelper VZHelper) error {
	list, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		LogError(fmt.Sprintf("An error occurred while getting the %s in namespace %s: %s\n", kind, namespace, err.Error()))
	}
	if len(list.Items) > 0 {
		LogMessage(fmt.Sprintf("%s in namespace: %s ...\n", kind, namespace))
		if err = createFile(list, namespace, fmt.Sprintf("%s.json", strings.ToLower(kind)), captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}
