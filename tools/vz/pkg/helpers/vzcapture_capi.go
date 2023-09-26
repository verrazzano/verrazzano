// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	v1Alpha1            = "v1alpha1"
	v1Alpha3            = "v1alpha3"
	v1Beta1             = "v1beta1"
	v1Beta2             = "v1beta2"
	addonsGroup         = "addons.cluster.x-k8s.io"
	bootstrapGroup      = "bootstrap.cluster.x-k8s.io"
	clusterGroup        = "cluster.x-k8s.io"
	clusterctlGroup     = "clusterctl.cluster.x-k8s.io"
	controlPlaneGroup   = "controlplane.cluster.x-k8s.io"
	infrastructureGroup = "infrastructure.cluster.x-k8s.io"
	ipamGroup           = "ipam.cluster.x-k8s.io"
	runtimeGroup        = "runtime.cluster.x-k8s.io"
)

type capiResource struct {
	GVR  schema.GroupVersionResource
	Kind string
}

// capiResources - resources that are not namespaced
var capiResources = []capiResource{
	{GVR: schema.GroupVersionResource{Group: runtimeGroup, Version: v1Alpha1, Resource: "extensionconfigs"}, Kind: "ExtensionConfig"},
}

// capiNamespacedResources - resources that are namespaced
var capiNamespacedResources = []capiResource{
	{GVR: schema.GroupVersionResource{Group: addonsGroup, Version: v1Beta1, Resource: "clusterresourcesetbindings"}, Kind: "ClusterResourceSetBinding"},
	{GVR: schema.GroupVersionResource{Group: addonsGroup, Version: v1Beta1, Resource: "clusterresourcesets"}, Kind: "ClusterResourceSet"},
	{GVR: schema.GroupVersionResource{Group: bootstrapGroup, Version: v1Alpha1, Resource: "ocneconfigs"}, Kind: "OCNEConfig"},
	{GVR: schema.GroupVersionResource{Group: bootstrapGroup, Version: v1Alpha1, Resource: "ocneconfigtemplates"}, Kind: "OCNEConfigTemplate"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "clusterclasses"}, Kind: "ClusterClass"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "clusters"}, Kind: "Cluster"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinedeployments"}, Kind: "MachineDeployment"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinehealthchecks"}, Kind: "MachineHealthCheck"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinepools"}, Kind: "MachinePool"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machines"}, Kind: "Machine"},
	{GVR: schema.GroupVersionResource{Group: clusterGroup, Version: v1Beta1, Resource: "machinesets"}, Kind: "MachineSet"},
	{GVR: schema.GroupVersionResource{Group: clusterctlGroup, Version: v1Alpha3, Resource: "providers"}, Kind: "Provider"},
	{GVR: schema.GroupVersionResource{Group: controlPlaneGroup, Version: v1Alpha1, Resource: "ocnecontrolplanes"}, Kind: "OCNEControlPlane"},
	{GVR: schema.GroupVersionResource{Group: controlPlaneGroup, Version: v1Alpha1, Resource: "ocnecontrolplanetemplates"}, Kind: "OCNEControlPlaneTemplate"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclusteridentities"}, Kind: "OCIClusterIdentity"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclusters"}, Kind: "OCICluster"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclustertemplates"}, Kind: "OCIClusterTemplate"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachinepools"}, Kind: "OCIMachinePool"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachines"}, Kind: "OCIMachine"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachinetemplates"}, Kind: "OCIMachineTemplate"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedclusters"}, Kind: "OCIManagedCluster"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedclustertemplates"}, Kind: "OCIManagedClusterTemplate"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedcontrolplanes"}, Kind: "OCIManagedControlPlane"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedcontrolplanetemplates"}, Kind: "OCIManagedControlPlaneTemplate"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedmachinepools"}, Kind: "OCIManagedMachinePool"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedmachinepooltemplates"}, Kind: "OCIManagedMachinePoolTemplate"},
	{GVR: schema.GroupVersionResource{Group: ipamGroup, Version: v1Alpha1, Resource: "ipaddressclaims"}, Kind: "IPAddressClaim"},
	{GVR: schema.GroupVersionResource{Group: ipamGroup, Version: v1Alpha1, Resource: "ipaddresses"}, Kind: "IPAddress"},
}

// CaptureGlobalCapiResources captures global resources related to ClusterAPI
func CaptureGlobalCapiResources(dynamicClient dynamic.Interface, captureDir string, vzHelper VZHelper) error {
	for _, resource := range capiResources {
		if err := captureGlobalResource(dynamicClient, resource.GVR, resource.Kind, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureCapiNamespacedResources captures resources related to ClusterAPI
func captureCapiNamespacedResources(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	for _, resource := range capiNamespacedResources {
		if err := captureNamespacedResource(dynamicClient, resource.GVR, resource.Kind, namespace, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

func captureGlobalResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, kind string, captureDir string, vzHelper VZHelper) error {
	list, err := dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		LogError(fmt.Sprintf("An error occurred while listing cluster resource %s: %s\n", kind, err.Error()))
		return nil
	}
	if len(list.Items) > 0 {
		LogMessage(fmt.Sprintf("%s in global namespace ...\n", kind))
		if err = createFile(list, "", fmt.Sprintf("%s.%s.json", strings.ToLower(kind), strings.ToLower(gvr.Group)), captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

func captureNamespacedResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, kind string, namespace, captureDir string, vzHelper VZHelper) error {
	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		LogError(fmt.Sprintf("An error occurred while listing %s in namespace %s: %s\n", kind, namespace, err.Error()))
		return nil
	}
	if len(list.Items) > 0 {
		LogMessage(fmt.Sprintf("%s in namespace: %s ...\n", kind, namespace))
		if err = createFile(list, namespace, fmt.Sprintf("%s.%s.json", strings.ToLower(kind), strings.ToLower(gvr.Group)), captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}
