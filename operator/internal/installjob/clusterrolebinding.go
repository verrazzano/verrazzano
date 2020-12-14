// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/verrazzano/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewClusterRoleBinding returns a cluster role binding resource for installing Verrazzano
// vz - pointer to verrazzano resource
// name - name of the clusterrolebinding resource
// saName - name of service account resource
func NewClusterRoleBinding(vz *installv1alpha1.Verrazzano, name string, saName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: vz.Labels,
			// Set owner reference here instead of calling controllerutil.SetControllerReference
			// which does not allow cluster-scoped resources.
			// This reference will result in the clusterrolebinding resource being deleted
			// when the verrazzano CR is deleted.
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: vz.APIVersion,
					Kind:       vz.Kind,
					Name:       vz.Name,
					UID:        vz.UID,
					Controller: func() *bool {
						flag := true
						return &flag
					}(),
				},
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: vz.Namespace,
			},
		},
	}
}
