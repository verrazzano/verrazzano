package controllers

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Info used to mutate binding
type bindingInfo struct {
	vmc                     *clustersv1alpha1.VerrazzanoManagedCluster
	roleBindingName         string
	roleName                string
	serviceAccountName      string
	serviceAccountNamespace string
}

// reconcileManagedRoleBinding reconciles the ClusterRoleBinding that binds the service account used by the managed cluster
// to the role containing the permission
func (r *VerrazzanoManagedClusterReconciler) reconcileManagedRoleBinding(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	bindingAndRoleName := generateManagedResourceName(vmc.Name)
	var binding rbacv1.ClusterRoleBinding
	binding.Namespace = vmc.Namespace
	binding.Name = bindingAndRoleName

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &binding, func() error {
		bindingInfo{
			vmc:                     vmc,
			roleBindingName:         bindingAndRoleName,
			roleName:                bindingAndRoleName,
			serviceAccountName:      vmc.Spec.ServiceAccount,
			serviceAccountNamespace: vmc.Namespace,
		}.mutateBinding(&binding)
		return controllerutil.SetControllerReference(vmc, &binding, r.Scheme)
	})
	return err
}

// mutateBinding mutes the ClusterRoleBinding to ensure it has the valid params
func (b bindingInfo) mutateBinding(binding *rbacv1.ClusterRoleBinding) {
	binding.ObjectMeta = metav1.ObjectMeta{
		Name:   b.roleBindingName,
		Labels: b.vmc.Labels,
		// Set owner reference here instead of calling controllerutil.SetControllerReference
		// which does not allow cluster-scoped resources.
		// This reference will result in the clusterrolebinding resource being deleted
		// when the verrazzano CR is deleted.
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: b.vmc.APIVersion,
				Kind:       b.vmc.Kind,
				Name:       b.vmc.Name,
				UID:        b.vmc.UID,
				Controller: func() *bool {
					flag := true
					return &flag
				}(),
			},
		},
	}
	binding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     b.roleName,
	}
	binding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      b.serviceAccountName,
			Namespace: b.serviceAccountNamespace,
		},
	}
}
