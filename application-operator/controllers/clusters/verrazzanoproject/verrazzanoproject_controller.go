// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vztime "github.com/verrazzano/verrazzano/pkg/time"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	projectAdminRole            = "verrazzano-project-admin"
	projectAdminK8sRole         = "admin"
	projectAdminGroupTemplate   = "verrazzano-project-%s-admins"
	projectMonitorRole          = "verrazzano-project-monitor"
	projectMonitorK8sRole       = "view"
	projectMonitorGroupTemplate = "verrazzano-project-%s-monitors"
	finalizerName               = "project.verrazzano.io"
	keyPolicyName               = "policy-name"
	keyNamespace                = "namespace"
	managedClusterRole          = "verrazzano-managed-cluster"
)

// Reconciler reconciles a VerrazzanoProject object
type Reconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	AgentChannel chan clusters.StatusUpdateMessage
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(
				vztime.SecsToDuration(vzconst.ControllerBaseDelay),
				vztime.SecsToDuration(vzconst.ControllerMaxDelay)),
		}).
		For(&clustersv1alpha1.VerrazzanoProject{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoProject resource.
// It fetches its namespaces if the VerrazzanoProject is in the verrazzano-mc namespace
// and create namespaces in the local cluster.
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("verrazzanoproject", req.NamespacedName)
	var vp clustersv1alpha1.VerrazzanoProject
	ctx := context.Background()
	logger.Info("Fetching VerrazzanoProject")
	err := r.Get(ctx, req.NamespacedName, &vp)
	if err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		logger.Error(err, "Failed to fetch VerrazzanoProject")
		return reconcile.Result{}, err
	}

	// Check if the project is being deleted
	if !vp.ObjectMeta.DeletionTimestamp.IsZero() {
		// If finalizer is present, delete the network policies in the project namespaces
		if vzstring.SliceContainsString(vp.ObjectMeta.Finalizers, finalizerName) {
			logger.Info("Deleting all network policies for project")
			if err := r.deleteNetworkPolicies(ctx, &vp, nil, logger); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.deleteRoleBindings(ctx, &vp, logger); err != nil {
				return reconcile.Result{}, err
			}
			// Remove the finalizer and update the Verrazzano resource if the deletion has finished.
			vp.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(vp.ObjectMeta.Finalizers, finalizerName)
			err := r.Update(ctx, &vp)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer if not already added
	if !vzstring.SliceContainsString(vp.ObjectMeta.Finalizers, finalizerName) {
		vp.ObjectMeta.Finalizers = append(vp.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, &vp); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Use OperationResultCreated by default since we don't really know what happened to individual resources
	opResult := controllerutil.OperationResultCreated
	err = r.syncAll(ctx, vp, logger)
	if err != nil {
		opResult = controllerutil.OperationResultNone
	}

	// Update the cluster status
	_, statusErr := r.updateStatus(ctx, &vp, opResult, err)
	if statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	// Update the VerrazzanoProject state
	oldState := clusters.SetEffectiveStateIfChanged(vp.Spec.Placement, &vp.Status)
	if oldState != vp.Status.State {
		stateErr := r.Status().Update(ctx, &vp)
		if stateErr != nil {
			return ctrl.Result{}, stateErr
		}
	}

	// if an error occurred in createOrUpdate, return that error with a requeue
	// even if update status succeeded
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: clusters.GetRandomRequeueDelay()}, err
	}
	return ctrl.Result{}, nil
}

// Sync all the project resources, return immediately with error if failure
func (r *Reconciler) syncAll(ctx context.Context, vp clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	err := r.createOrUpdateNamespaces(ctx, vp, logger)
	if err != nil {
		return err
	}

	// Sync the network policies
	err = r.syncNetworkPolices(ctx, &vp, logger)
	if err != nil {
		return err
	}
	return nil
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
				return err
			}

			if err = r.createOrUpdateRoleBindings(ctx, nsTemplate.Metadata.Name, vp, logger); err != nil {
				return err
			}

			if err = r.deleteRoleBindings(ctx, nil, logger); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Reconciler) mutateNamespace(nsTemplate clustersv1alpha1.NamespaceTemplate, namespace *corev1.Namespace) {
	namespace.Annotations = nsTemplate.Metadata.Annotations
	namespace.Spec = nsTemplate.Spec

	// Add Verrazzano generated labels if not already present
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

	// get the default binding subjects
	adminSubjects, monitorSubjects := r.getDefaultRoleBindingSubjects(vp)

	// override defaults if specified in the project
	if len(vp.Spec.Template.Security.ProjectAdminSubjects) > 0 {
		adminSubjects = vp.Spec.Template.Security.ProjectAdminSubjects
	}
	if len(vp.Spec.Template.Security.ProjectMonitorSubjects) > 0 {
		monitorSubjects = vp.Spec.Template.Security.ProjectMonitorSubjects
	}

	// create two role bindings, one for the project admin role and one for the k8s admin role
	if len(adminSubjects) > 0 {
		rb := newRoleBinding(namespace, projectAdminRole, adminSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
		rb = newRoleBinding(namespace, projectAdminK8sRole, adminSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
	}

	// create two role bindings, one for the project monitor role and one for the k8s monitor role
	if len(monitorSubjects) > 0 {
		rb := newRoleBinding(namespace, projectMonitorRole, monitorSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
		rb = newRoleBinding(namespace, projectMonitorK8sRole, monitorSubjects)
		if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
			return err
		}
	}

	// create role binding for each managed cluster to limit resource access to admin cluster
	for _, cluster := range vp.Spec.Placement.Clusters {
		if cluster.Name != constants.DefaultClusterName {
			rb := newRoleBindingManagedCluster(namespace, cluster.Name)
			if err := r.createOrUpdateRoleBinding(ctx, rb, logger); err != nil {
				return err
			}
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

// updateStatus updates the status of a VerrazzanoProject
func (r *Reconciler) updateStatus(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "VerrazzanoProject")
	updateFunc := func() error { return r.Status().Update(ctx, vp) }
	return clusters.UpdateStatus(vp, &vp.Status, vp.Spec.Placement, newCondition, clusterName,
		r.AgentChannel, updateFunc)
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

// newRoleBinding returns a populated RoleBinding struct for a given managed cluster
func newRoleBindingManagedCluster(namespace string, name string) *rbacv1.RoleBinding {
	clusterNameRef := generateRoleBindingManagedClusterRef(name)
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      clusterNameRef,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     managedClusterRole,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      clusterNameRef,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		},
	}
}
func generateRoleBindingManagedClusterRef(name string) string {
	return fmt.Sprintf("verrazzano-cluster-%s", name)
}

// getDefaultRoleBindingSubjects returns the default binding subjects for project admin/monitor roles
func (r *Reconciler) getDefaultRoleBindingSubjects(vp clustersv1alpha1.VerrazzanoProject) ([]rbacv1.Subject, []rbacv1.Subject) {
	adminSubjects := []rbacv1.Subject{{
		Kind: "Group",
		Name: fmt.Sprintf(projectAdminGroupTemplate, vp.Name),
	}}
	monitorSubjects := []rbacv1.Subject{{
		Kind: "Group",
		Name: fmt.Sprintf(projectMonitorGroupTemplate, vp.Name),
	}}
	return adminSubjects, monitorSubjects
}

// syncNetworkPolices syncs the NetworkPolicies specified in the project
func (r *Reconciler) syncNetworkPolices(ctx context.Context, project *clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	// Create or update policies that are in the project spec
	// The project webhook validates that the network policies use project namespaces
	desiredPolicySet := make(map[string]bool)
	for i, policyTemplate := range project.Spec.Template.NetworkPolicies {
		desiredPolicySet[policyTemplate.Metadata.Namespace+policyTemplate.Metadata.Name] = true
		_, err := r.createOrUpdateNetworkPolicy(ctx, &project.Spec.Template.NetworkPolicies[i])
		if err != nil {
			return err
		}
	}
	// Delete policies in this namespace that should not exist
	return r.deleteNetworkPolicies(ctx, project, desiredPolicySet, logger)
}

// createOrUpdateNetworkPolicy creates or updates the network polices in the project
func (r *Reconciler) createOrUpdateNetworkPolicy(ctx context.Context, desiredPolicy *clustersv1alpha1.NetworkPolicyTemplate) (controllerutil.OperationResult, error) {
	var policy netv1.NetworkPolicy
	policy.Namespace = desiredPolicy.Metadata.Namespace
	policy.Name = desiredPolicy.Metadata.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &policy, func() error {
		desiredPolicy.Metadata.DeepCopyInto(&policy.ObjectMeta)
		desiredPolicy.Spec.DeepCopyInto(&policy.Spec)
		return nil
	})
}

func (r *Reconciler) deleteRoleBindings(ctx context.Context, project *clustersv1alpha1.VerrazzanoProject, logger logr.Logger) error {
	// Get the list of VerrazzanoProject resources
	vpList := clustersv1alpha1.VerrazzanoProjectList{}
	if err := r.List(ctx, &vpList, client.InNamespace(constants.VerrazzanoMultiClusterNamespace)); err != nil {
		return err
	}

	// Create map of expected namespace/cluster pairs for rolebindings
	expectedPairs := make(map[string]bool)
	for _, vp := range vpList.Items {
		if project != nil && project.Name == vp.Name {
			continue
		}
		for _, ns := range vp.Spec.Template.Namespaces {
			for _, cluster := range vp.Spec.Placement.Clusters {
				expectedPairs[ns.Metadata.Name+cluster.Name] = true
			}
		}
	}

	// Get the list of VerrazzanoManagedCluster resources
	vmcList := v1alpha1.VerrazzanoManagedClusterList{}
	err := r.List(ctx, &vmcList, client.InNamespace(constants.VerrazzanoMultiClusterNamespace))
	if err != nil {
		return err
	}

	for _, vmc := range vmcList.Items {
		for _, vp := range vpList.Items {
			for _, ns := range vp.Spec.Template.Namespaces {
				// rolebinding is expected for this namespace/cluster pairing
				// so nothing to delete
				if _, ok := expectedPairs[ns.Metadata.Name+vmc.Name]; ok {
					continue
				}
				// rolebinding is not expected for this namespace/cluster pairing
				objectKey := types.NamespacedName{
					Namespace: ns.Metadata.Name,
					Name:      generateRoleBindingManagedClusterRef(vmc.Name),
				}
				rb := rbacv1.RoleBinding{}
				if err := r.Get(ctx, objectKey, &rb); err != nil {
					continue
				}
				// This is an orphaned rolebinding so we delete it
				logger.Info("Deleting rolebinding for project", "namespace", rb.ObjectMeta.Namespace, "roleName", rb.ObjectMeta.Name)
				if err := r.Delete(ctx, &rb); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// deleteNetworkPolicies deletes policies that exist in the project namespaces, but are not defined in the project spec
func (r *Reconciler) deleteNetworkPolicies(ctx context.Context, project *clustersv1alpha1.VerrazzanoProject, desiredPolicySet map[string]bool, logger logr.Logger) error {
	for _, ns := range project.Spec.Template.Namespaces {
		// Get the list of policies in the namespace
		policies := netv1.NetworkPolicyList{}
		if err := r.List(ctx, &policies, client.InNamespace(ns.Metadata.Name)); err != nil {
			return err
		}
		// Loop through the policies found in the namespace
		for pi, policy := range policies.Items {
			if desiredPolicySet != nil {
				// Don't delete policy if it should be in the namespace
				if _, ok := desiredPolicySet[policy.Namespace+policy.Name]; ok {
					continue
				}
			}

			// Found a policy in the namespace that is not specified in the project.  Delete it
			if err := r.Delete(ctx, &policies.Items[pi], &client.DeleteOptions{}); err != nil {
				logger.Error(err, "Unable to delete NetworkPolicy during cleanup of project",
					keyNamespace, policy.Namespace, keyPolicyName, policy.Name)
			}
		}
	}
	return nil
}
