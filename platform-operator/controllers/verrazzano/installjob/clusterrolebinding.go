// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getObjectMetaForName returns a metav1.ObjectMeta data filled in with name, labels, and owner references
func getObjectMetaForName(vz *installv1alpha1.Verrazzano, name, resourceVersion string) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
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
	}
	if len(resourceVersion) > 0 {
		meta.ResourceVersion = resourceVersion
	}
	return meta
}

// getRoleRefForName returns a RoleRef for a ClusterRole with the given name
func getRoleRefForName(name string) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     name,
	}
}

// NewClusterRoleBinding returns a cluster role binding for the Verrazzano system account
// vz - pointer to verrazzano resource
// name - name of the clusterrolebinding resource
// saName - name of service account resource
func NewClusterRoleBinding(vz *installv1alpha1.Verrazzano, name string, saName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: getObjectMetaForName(vz, name, ""),
		RoleRef:    getRoleRefForName("cluster-admin"),
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: vz.Namespace,
			},
		},
	}
}

// NewClusterRoleBindingWithSubjects returns a cluster role binding with the given parameters
// vz - pointer to verrazzano resource
// name - name of the clusterrolebinding resource
// roleName - name of the clusterrole to bind to
// subjects - slice of subjects to bind to the role
func NewClusterRoleBindingWithSubjects(vz *installv1alpha1.Verrazzano, bindingName string, roleName string, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: getObjectMetaForName(vz, bindingName, ""),
		RoleRef:    getRoleRefForName(roleName),
		Subjects:   subjects,
	}
}

// GetClusterRoleBindingForPatch - updates an existing ClusterRoleBinding with the provided subjects
// vz - pointer to verrazzano resource
// existing - pointer to existing ClusterRoleBinding
// subjects - slice of Subjects to replace the existing subjects
func GetClusterRoleBindingForPatch(vz *installv1alpha1.Verrazzano, existing *rbacv1.ClusterRoleBinding, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: getObjectMetaForName(vz, existing.ObjectMeta.Name, existing.ObjectMeta.ResourceVersion),
		// can't update the RoleRef for an existing binding
		Subjects: subjects,
	}
}
