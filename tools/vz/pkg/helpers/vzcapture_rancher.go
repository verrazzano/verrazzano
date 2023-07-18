// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	cattleV1          = "v1"
	cattleV3          = "v3"
	cattleV1Alpha1    = "v1alpha1"
	mgmtGroup         = "management.cattle.io"
	catalogGroup      = "catalog.cattle.io"
	provisioningGroup = "provisioning.cattle.io"
	fleetGroup        = "fleet.cattle.io"
)

type rancherResource struct {
	GVR  schema.GroupVersionResource
	Kind string
}

// rancherResources - resources that are not namespaced
var rancherResources = []rancherResource{
	{GVR: schema.GroupVersionResource{Group: catalogGroup, Version: cattleV1, Resource: "clusterrepos"}, Kind: "ClusterRepo"},
	{GVR: schema.GroupVersionResource{Group: mgmtGroup, Version: cattleV3, Resource: "catalogs"}, Kind: "Catalog"},
	{GVR: schema.GroupVersionResource{Group: mgmtGroup, Version: cattleV3, Resource: "kontainerdrivers"}, Kind: "KontainerDriver"},
	{GVR: schema.GroupVersionResource{Group: mgmtGroup, Version: cattleV3, Resource: "clusters"}, Kind: "Cluster"},
}

// rancherNamespacedResources - resources that are namespaced
var rancherNamespacedResources = []rancherResource{
	{GVR: schema.GroupVersionResource{Group: fleetGroup, Version: cattleV1Alpha1, Resource: "bundles"}, Kind: "Bundle"},
	{GVR: schema.GroupVersionResource{Group: fleetGroup, Version: cattleV1Alpha1, Resource: "bundledeployments"}, Kind: "BundleDeployment"},
	{GVR: schema.GroupVersionResource{Group: fleetGroup, Version: cattleV1Alpha1, Resource: "clusters"}, Kind: "Cluster"},
	{GVR: schema.GroupVersionResource{Group: fleetGroup, Version: cattleV1Alpha1, Resource: "clustergroups"}, Kind: "ClusterGroup"},
	{GVR: schema.GroupVersionResource{Group: fleetGroup, Version: cattleV1Alpha1, Resource: "clusterregistrations"}, Kind: "ClusterRegistration"},
	{GVR: schema.GroupVersionResource{Group: provisioningGroup, Version: cattleV1, Resource: "clusters"}, Kind: "Cluster"},
	{GVR: schema.GroupVersionResource{Group: mgmtGroup, Version: cattleV3, Resource: "nodes"}, Kind: "Node"},
}

// CaptureGlobalRancherResources captures global resources related to ClusterAPI
func CaptureGlobalRancherResources(dynamicClient dynamic.Interface, captureDir string, vzHelper VZHelper) error {
	for _, resource := range rancherResources {
		if err := captureGlobalResource(dynamicClient, resource.GVR, resource.Kind, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureRancherNamespacedResources captures resources that are namespaced
func captureRancherNamespacedResources(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	for _, resource := range rancherNamespacedResources {
		if err := captureNamespacedResource(dynamicClient, resource.GVR, resource.Kind, namespace, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}
