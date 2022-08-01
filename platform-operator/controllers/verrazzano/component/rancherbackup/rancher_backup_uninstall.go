// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext) error {
	ctx.Log().Infof("Cleaning up rancher-backup cluster-role-binding and finalizers")

	clusterRbList := rbacv1.ClusterRoleBindingList{}
	if err := ctx.Client().List(context.TODO(), &clusterRbList, &client.ListOptions{}); err != nil {
		return err
	}
	for i, crb := range clusterRbList.Items {
		if crb.Name == ComponentName {
			if err := ctx.Client().Delete(context.TODO(), &clusterRbList.Items[i]); err != nil {
				return err
			}
			ctx.Log().Oncef("%v cluster role binding deleted successfully", crb.RoleRef.Name)
			return nil
		}
	}

	res := resource.Resource{
		Name:   ComponentNamespace,
		Client: ctx.Client(),
		Object: &corev1.Namespace{},
		Log:    ctx.Log(),
	}
	// Remove finalizers from the cattle-resources-system namespace to avoid hanging namespace deletion
	// and delete the namespace
	return res.RemoveFinalizersAndDelete()
}
