// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"context"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteMCResources deletes multicluster related resources
func DeleteMCResources(componentCtx componentspi.ComponentContext) error {
	// Check if this is not managed cluster
	managed, err := isManagedCluster(componentCtx.Client(), componentCtx.Log())
	if err != nil {
		return err
	}

	projects := vzappclusters.VerrazzanoProjectList{}
	if err := componentCtx.Client().List(context.TODO(), &projects, &client.ListOptions{Namespace: vzconst.VerrazzanoMultiClusterNamespace}); err != nil && !meta.IsNoMatchError(err) {
		return componentCtx.Log().ErrorfNewErr("Failed listing MC projects: %v", err)
	}
	// Delete MC rolebindings for each project
	for _, p := range projects.Items {
		if err := deleteManagedClusterRoleBindings(componentCtx.Client(), p, componentCtx.Log()); err != nil {
			return err
		}
	}

	componentCtx.Log().Oncef("Deleting all VMC resources")
	vmcList := clustersapi.VerrazzanoManagedClusterList{}
	if err := componentCtx.Client().List(context.TODO(), &vmcList, &client.ListOptions{}); err != nil && !meta.IsNoMatchError(err) {
		return componentCtx.Log().ErrorfNewErr("Failed listing VMCs: %v", err)
	}

	for i, vmc := range vmcList.Items {
		// Delete the VMC ServiceAccount (since managed cluster role bindings associated to it should now be deleted)
		vmcSA := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Namespace: vmc.Namespace, Name: vmc.Spec.ServiceAccount},
		}
		if err := componentCtx.Client().Delete(context.TODO(), &vmcSA); err != nil {
			return componentCtx.Log().ErrorfNewErr("Failed to delete VMC service account %s/%s, %v", vmc.Namespace, vmc.Spec.ServiceAccount, err)
		}
		if err := componentCtx.Client().Delete(context.TODO(), &vmcList.Items[i]); err != nil {
			return componentCtx.Log().ErrorfNewErr("Failed to delete VMC %s/%s, %v", vmc.Namespace, vmc.Name, err)
		}
	}

	// Delete VMC namespace only if there are no projects
	if len(projects.Items) == 0 {
		componentCtx.Log().Oncef("Deleting %s namespace", vzconst.VerrazzanoMultiClusterNamespace)
		if err := DeleteNamespace(componentCtx.Client(), componentCtx.Log(), vzconst.VerrazzanoMultiClusterNamespace); err != nil {
			return err
		}
	}

	// Delete secrets on managed cluster.  Don't delete MC agent secret until the end since it tells us this is MC install
	if managed {
		if err := DeleteSecret(componentCtx.Client(), componentCtx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret); err != nil {
			return err
		}
		if err := DeleteSecret(componentCtx.Client(), componentCtx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret); err != nil {
			return err
		}
		if err := DeleteSecret(componentCtx.Client(), componentCtx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret); err != nil {
			return err
		}
	}

	return nil
}

// deleteManagedClusterRoleBindings deletes the managed cluster rolebindings from each namespace
// governed by the given project
func deleteManagedClusterRoleBindings(cli client.Client, project vzappclusters.VerrazzanoProject, _ vzlog.VerrazzanoLogger) error {
	for _, projectNSTemplate := range project.Spec.Template.Namespaces {
		rbList := rbacv1.RoleBindingList{}
		if err := cli.List(context.TODO(), &rbList, &client.ListOptions{Namespace: projectNSTemplate.Metadata.Name}); err != nil {
			return err
		}
		for i, rb := range rbList.Items {
			if rb.RoleRef.Name == "verrazzano-managed-cluster" {
				if err := cli.Delete(context.TODO(), &rbList.Items[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// isManagedCluster returns true if this is a managed cluster
func isManagedCluster(cli client.Client, log vzlog.VerrazzanoLogger) (bool, error) {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: vzconst.VerrazzanoSystemNamespace,
		Name:      vzconst.MCAgentSecret,
	}

	// Get the MC agent secret and return if not found
	if err := cli.Get(context.TODO(), secretNsn, &secret); err != nil {
		if errors.IsNotFound(err) {
			log.Once("Determined that this is not a managed cluster")
			return false, nil
		}
		return false, log.ErrorfNewErr("Failed to fetch the multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
	}
	return true, nil
}
