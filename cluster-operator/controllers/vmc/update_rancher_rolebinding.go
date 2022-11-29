// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	APIGroupRancherManagement        = "management.cattle.io"
	APIGroupVersionRancherManagement = "v3"
	ClusterRoleTemplateBindingKind   = "ClusterRoleTemplateBinding"

	ClusterRoleTemplateBindingAttributeClusterName      = "clusterName"
	ClusterRoleTemplateBindingAttributeUserName         = "userName"
	ClusterRoleTemplateBindingAttributeRoleTemplateName = "roleTemplateName"
)

// UpdateRancherClusterRoleBindingTemplate creates a new ClusterRoleBindingTemplate for the given VMC
// to grant the Verrazzano cluster user the correct permissions on the managed cluster
func (r *VerrazzanoManagedClusterReconciler) UpdateRancherClusterRoleBindingTemplate(vmc *v1alpha1.VerrazzanoManagedCluster) error {
	if vmc == nil {
		r.log.Debugf("Empty VMC, no ClusterRoleBindingTemplate created")
		return nil
	}
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		r.log.Debugf("Waiting to create ClusterRoleBindingTemplate for cluster %s, Rancher ClusterID not found in the VMC status", vmc.GetName())
		return nil
	}

	name := fmt.Sprintf("crtb-verrazzano-cluster-%s", clusterID)
	nsn := types.NamespacedName{Name: name, Namespace: clusterID}
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    ClusterRoleTemplateBindingKind,
	})
	resource.SetName(nsn.Name)
	resource.SetNamespace(nsn.Namespace)
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &resource, func() error {
		data := resource.UnstructuredContent()
		data[ClusterRoleTemplateBindingAttributeClusterName] = clusterID
		data[ClusterRoleTemplateBindingAttributeUserName] = vzconst.VerrazzanoClusterRancherUsername
		data[ClusterRoleTemplateBindingAttributeRoleTemplateName] = vzconst.VerrazzanoClusterRancherName
		return nil
	})
	if err != nil {
		return r.log.ErrorfThrottledNewErr("Failed configuring %s %s: %s", ClusterRoleTemplateBindingKind, nsn.Name, err.Error())
	}
	return nil
}
