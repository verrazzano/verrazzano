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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	APIGroupRancherManagement        = "management.cattle.io"
	APIGroupVersionRancherManagement = "v3"
	ClusterRoleTemplateBindingKind   = "ClusterRoleTemplateBinding"
	UserListKind                     = "UserList"
	UserKind                         = "User"

	ClusterRoleTemplateBindingAttributeClusterName      = "clusterName"
	ClusterRoleTemplateBindingAttributeUserName         = "userName"
	ClusterRoleTemplateBindingAttributeRoleTemplateName = "roleTemplateName"

	UserUsernameAttribute = "username"
)

// updateRancherClusterRoleBindingTemplate creates a new ClusterRoleBindingTemplate for the given VMC
// to grant the Verrazzano cluster user the correct permissions on the managed cluster
func (r *VerrazzanoManagedClusterReconciler) updateRancherClusterRoleBindingTemplate(vmc *v1alpha1.VerrazzanoManagedCluster) error {
	if vmc == nil {
		r.log.Debugf("Empty VMC, no ClusterRoleBindingTemplate created")
		return nil
	}

	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		r.log.Progressf("Waiting to create ClusterRoleBindingTemplate for cluster %s, Rancher ClusterID not found in the VMC status", vmc.GetName())
		return nil
	}

	userID, err := r.getVZClusterUserID()
	if err != nil {
		return err
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
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &resource, func() error {
		data := resource.UnstructuredContent()
		data[ClusterRoleTemplateBindingAttributeClusterName] = clusterID
		data[ClusterRoleTemplateBindingAttributeUserName] = userID
		data[ClusterRoleTemplateBindingAttributeRoleTemplateName] = vzconst.VerrazzanoClusterRancherName
		return nil
	})
	if err != nil {
		return r.log.ErrorfThrottledNewErr("Failed configuring %s %s: %s", ClusterRoleTemplateBindingKind, nsn.Name, err.Error())
	}
	return nil
}

// getVZClusterUserID returns the Rancher-generated user ID for the Verrazzano cluster user
func (r *VerrazzanoManagedClusterReconciler) getVZClusterUserID() (string, error) {
	usersList := unstructured.UnstructuredList{}
	usersList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    UserListKind,
	})
	err := r.Client.List(context.TODO(), &usersList, &client.ListOptions{})
	if err != nil {
		return "", r.log.ErrorfNewErr("Failed to list Rancher Users: %v", err)
	}

	for _, user := range usersList.Items {
		userData := user.UnstructuredContent()
		if userData[UserUsernameAttribute] == vzconst.VerrazzanoClusterRancherUsername {
			return user.GetName(), nil
		}
	}
	return "", r.log.ErrorfNewErr("Failed to find a Rancher user with username %s", vzconst.VerrazzanoClusterRancherUsername)
}
