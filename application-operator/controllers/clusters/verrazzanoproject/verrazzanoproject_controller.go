// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	projectAdminRole      = "verrazzano-project-admin"
	projectAdminK8sRole   = "admin"
	projectMonitorRole    = "verrazzano-project-monitor"
	projectMonitorK8sRole = "view"
)

// Reconciler reconciles a VerrazzanoProject object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoProject{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoProject resource.
// It fetches its namespaces if the VerrazzanoProject is in the verrazzano-mc namespace
// and create namespaces in the local cluster.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("verrazzanoproject", req.NamespacedName)
	var vp clustersv1alpha1.VerrazzanoProject
	result := reconcile.Result{}
	ctx := context.Background()
	logger.Info("Fetching VerrazzanoProject")
	err := r.Get(ctx, req.NamespacedName, &vp)
	if err != nil {
		logger.Error(err, "Failed to fetch VerrazzanoProject")
		return result, client.IgnoreNotFound(err)
	}

	err = r.createOrUpdateNamespaces(ctx, vp, logger)
	return result, err
}

func (r *Reconciler) createOrUpdateNamespaces(ctx context.Context, vp clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	if vp.Namespace == constants.VerrazzanoMultiClusterNamespace {
		for _, nsTemplate := range vp.Spec.Template.Namespaces {
			logger.Info("create or update with underlying namespace", "namespace", nsTemplate.Metadata.Name)
			var namespace corev1.Namespace
			namespace.Name = nsTemplate.Metadata.Name

			opResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, &namespace, func() error {
				r.mutateNamespace(nsTemplate, &namespace)
				return nil
			})
			if err != nil {
				logger.Error(err, "create or update namespace failed", "namespace", nsTemplate.Metadata.Name, "opResult", opResult)
			}

			if err = r.createOrUpdateRoleBindings(ctx, nsTemplate.Metadata.Name, vp, logger); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Reconciler) mutateNamespace(nsTemplate clustersv1alpha1.NamespaceTemplate, namespace *corev1.Namespace) {
	namespace.Annotations = nsTemplate.Metadata.Annotations
	namespace.Spec = nsTemplate.Spec

	// Add verrazzano generated labels if not already present
	if namespace.Labels == nil {
		namespace.Labels = map[string]string{}
	}

	// Apply the standard Verrazzano labels
	namespace.Labels[constants.LabelVerrazzanoManaged] = constants.LabelVerrazzanoManagedDefault
	namespace.Labels[constants.LabelIstioInjection] = constants.LabelIstioInjectionDefault

	// Apply user specified labels, which may override standard Verrazzano labels
	for label, value := range nsTemplate.Metadata.Labels {
		namespace.Labels[label] = value
	}
}

// createOrUpdateRoleBindings creates project role bindings if there are security subjects specified in
// the project spec
func (r *Reconciler) createOrUpdateRoleBindings(ctx context.Context, namespace string, vp clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	logger.Info("Create or update role bindings", "namespace", namespace)

	// if there are any project admin subjects, create two role bindings, one for the project admin role and
	// one for the k8s admin role
	if len(vp.Spec.Template.Security.ProjectAdminSubjects) > 0 {
		rb := newRoleBinding(namespace, projectAdminRole, vp.Spec.Template.Security.ProjectAdminSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
		rb = newRoleBinding(namespace, projectAdminK8sRole, vp.Spec.Template.Security.ProjectAdminSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
	}
	// if there are any project monitor subjects, create two role bindings, one for the project monitor role and
	// one for the k8s monitor role
	if len(vp.Spec.Template.Security.ProjectMonitorSubjects) > 0 {
		rb := newRoleBinding(namespace, projectMonitorRole, vp.Spec.Template.Security.ProjectMonitorSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
		rb = newRoleBinding(namespace, projectMonitorK8sRole, vp.Spec.Template.Security.ProjectMonitorSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
	}

	return nil
}

// createOrUpdateRoleBinding creates or updates a role binding
func (r *Reconciler) createOrUpdateRoleBinding(ctx context.Context, roleBinding *rbacv1.RoleBinding, logger logr.Logger) error {
	logger.Info("Create or update role binding", "roleName", roleBinding.ObjectMeta.Name)

	// deep copy the rolebinding so we can use the data in the mutate function
	rbCopy := roleBinding.DeepCopy()

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		// overwrite the roleref and subjects in case they changed out of band
		roleBinding.RoleRef = rbCopy.RoleRef
		roleBinding.Subjects = rbCopy.Subjects
		return nil
	})
	if err != nil {
		logger.Error(err, "Unable to create or update rolebinding", "roleName", roleBinding.ObjectMeta.Name)
		return err
	}

	return err
}

// newRoleBinding returns a populated RoleBinding struct
func newRoleBinding(namespace string, roleName string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      roleName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: subjects,
	}
}
