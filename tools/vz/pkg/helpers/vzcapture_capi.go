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
	{GVR: schema.GroupVersionResource{Group: clusterctlGroup, Version: v1Alpha3, Resource: "providers"}, Kind: "Providers"},
	{GVR: schema.GroupVersionResource{Group: controlPlaneGroup, Version: v1Alpha1, Resource: "ocnecontrolplanes"}, Kind: "OCNEControlPlanes"},
	{GVR: schema.GroupVersionResource{Group: controlPlaneGroup, Version: v1Alpha1, Resource: "ocnecontrolplanetemplates"}, Kind: "OCNEControlPlaneTemplates"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclusteridentities"}, Kind: "OCIClusterIdentities"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclusters"}, Kind: "OCIClusters"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ociclustertemplates"}, Kind: "OCIClusterTemplates"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachinepools"}, Kind: "OCIMachinePools"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachines"}, Kind: "OCIMachines"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimachinetemplates"}, Kind: "OCIMachineTemplates"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedclusters"}, Kind: "OCIManagedClusters"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedclustertemplates"}, Kind: "OCIManagedClusterTemplates"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedcontrolplanes"}, Kind: "OCIManagedControlPlanes"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedcontrolplanetemplates"}, Kind: "OCIManagedControlPlaneTemplates"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedmachinepools"}, Kind: "OCIManagedMachinePools"},
	{GVR: schema.GroupVersionResource{Group: infrastructureGroup, Version: v1Beta2, Resource: "ocimanagedmachinepooltemplates"}, Kind: "OCIManagedMachinePoolTemplates"},
	{GVR: schema.GroupVersionResource{Group: ipamGroup, Version: v1Alpha1, Resource: "ipaddressclaims"}, Kind: "IPAddressClaims"},
	{GVR: schema.GroupVersionResource{Group: ipamGroup, Version: v1Alpha1, Resource: "ipaddresses"}, Kind: "IPAddresses"},
	{GVR: schema.GroupVersionResource{Group: runtimeGroup, Version: v1Alpha1, Resource: "extensionconfigs"}, Kind: "ExtensionConfigs"},
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
	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
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
