// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "managedcluster.verrazzano.io"

// VerrazzanoManagedClusterReconciler reconciles a VerrazzanoManagedCluster object.
// The reconciler will create a ServiceAcount, RoleBinding, and a Secret which
// contains the kubeconfig to be used by the Multi-Cluster Agent to access the admin cluster.
type VerrazzanoManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    vzlog.VerrazzanoLogger
}

// bindingParams used to mutate the RoleBinding
type bindingParams struct {
	vmc                *clustersv1alpha1.VerrazzanoManagedCluster
	roleName           string
	serviceAccountName string
}

func (r *VerrazzanoManagedClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the  resource
	if ctx == nil {
		ctx = context.TODO()
	}
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	if err := r.Get(context.TODO(), req.NamespacedName, cr); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch VerrazzanoManagedCluster resource: %v", err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           cr.Name,
		Namespace:      cr.Namespace,
		ID:             string(cr.UID),
		Generation:     cr.Generation,
		ControllerName: "multicluster",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for VerrazzanoManagedCluster controller", err)
	}

	r.log = log
	log.Oncef("Reconciling Verrazzano resource %v", req.NamespacedName)
	res, err := r.doReconcile(ctx, log, cr)
	if vzctrl.ShouldRequeue(res) {
		return res, nil
	}

	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return newRequeueWithDelay(), nil
	}

	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return newRequeueWithDelay(), nil
	}

	// The resource has been reconciled.
	log.Oncef("Successfully reconciled VerrazzanoManagedCluster resource %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// Reconcile reconciles a VerrazzanoManagedCluster object
func (r *VerrazzanoManagedClusterReconciler) doReconcile(ctx context.Context, log vzlog.VerrazzanoLogger, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (ctrl.Result, error) {

	if !vmc.ObjectMeta.DeletionTimestamp.IsZero() {
		// Finalizer is present, so lets do the cluster deletion
		if vzstring.SliceContainsString(vmc.ObjectMeta.Finalizers, finalizerName) {
			if err := r.reconcileManagedClusterDelete(ctx, vmc); err != nil {
				return reconcile.Result{}, err
			}

			// Remove the finalizer and update the Verrazzano resource if the deletion has finished.
			log.Infof("Removing finalizer %s", finalizerName)
			vmc.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(vmc.ObjectMeta.Finalizers, finalizerName)
			err := r.Update(ctx, vmc)
			if err != nil && !errors.IsConflict(err) {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add our finalizer if not already added
	if !vzstring.SliceContainsString(vmc.ObjectMeta.Finalizers, finalizerName) {
		log.Infof("Adding finalizer %s", finalizerName)
		vmc.ObjectMeta.Finalizers = append(vmc.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, vmc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync the service account
	log.Debugf("Syncing the ServiceAccount for VMC %s", vmc.Name)
	err := r.syncServiceAccount(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the ServiceAccount", err, log)
		return newRequeueWithDelay(), err
	}

	log.Debugf("Syncing the RoleBinding for VMC %s", vmc.Name)
	_, err = r.syncManagedRoleBinding(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the RoleBinding", err, log)
		return newRequeueWithDelay(), err
	}

	log.Debugf("Syncing the Agent secret for VMC %s", vmc.Name)
	err = r.syncAgentSecret(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the agent secret", err, log)
		return newRequeueWithDelay(), err
	}

	log.Debugf("Syncing the Registration secret for VMC %s", vmc.Name)
	err = r.syncRegistrationSecret(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the registration secret", err, log)
		return newRequeueWithDelay(), err
	}

	log.Debugf("Syncing the Manifest secret for VMC %s", vmc.Name)
	err = r.syncManifestSecret(ctx, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the Manifest secret", err, log)
		return newRequeueWithDelay(), err
	}

	statusErr := r.updateStatusReady(ctx, vmc, "Ready")
	if statusErr != nil {
		log.Errorf("Failed to update status to ready for VMC %s: %v", vmc.Name, err)
	}

	if vmc.Status.PrometheusHost == "" {
		log.Infof("Managed cluster Prometheus Host not found in VMC Status for VMC %s. Waiting for VMC to be registered...", vmc.Name)
		return ctrl.Result{}, nil
	}

	log.Debugf("Syncing the prometheus scraper for VMC %s", vmc.Name)
	err = r.syncPrometheusScraper(ctx, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to setup the prometheus scraper for managed cluster", err, log)
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{Requeue: true, RequeueAfter: constants.ReconcileLoopRequeueInterval}, nil
}

func (r *VerrazzanoManagedClusterReconciler) syncServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Create or update the service account
	_, err := r.createOrUpdateServiceAccount(context.TODO(), vmc)
	if err != nil {
		return err
	}

	// Does the VerrazzanoManagedCluster object contain the service account name?
	saName := generateManagedResourceName(vmc.Name)
	if vmc.Spec.ServiceAccount != saName {
		r.log.Oncef("Updating ServiceAccount from %s to %s", vmc.Spec.ServiceAccount, saName)
		vmc.Spec.ServiceAccount = saName
		err = r.Update(context.TODO(), vmc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create or update the ServiceAccount for a VerrazzanoManagedCluster
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateServiceAccount(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (controllerutil.OperationResult, error) {
	var serviceAccount corev1.ServiceAccount
	serviceAccount.Namespace = vmc.Namespace
	serviceAccount.Name = generateManagedResourceName(vmc.Name)

	return controllerutil.CreateOrUpdate(ctx, r.Client, &serviceAccount, func() error {
		r.mutateServiceAccount(vmc, &serviceAccount)
		// This SetControllerReference call will trigger garbage collection i.e. the serviceAccount
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &serviceAccount, r.Scheme)
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster, serviceAccount *corev1.ServiceAccount) {
	serviceAccount.Name = generateManagedResourceName(vmc.Name)
}

// syncManagedRoleBinding syncs the RoleBinding that binds the service account used by the managed cluster
// to the role containing the permission
func (r *VerrazzanoManagedClusterReconciler) syncManagedRoleBinding(vmc *clustersv1alpha1.VerrazzanoManagedCluster) (controllerutil.OperationResult, error) {
	var roleBinding rbacv1.RoleBinding
	roleBinding.Namespace = vmc.Namespace
	roleBinding.Name = generateManagedResourceName(vmc.Name)

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &roleBinding, func() error {
		mutateBinding(&roleBinding, bindingParams{
			vmc:                vmc,
			roleName:           constants.MCClusterRole,
			serviceAccountName: vmc.Spec.ServiceAccount,
		})
		// This SetControllerReference call will trigger garbage collection i.e. the roleBinding
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &roleBinding, r.Scheme)
	})
}

// mutateBinding mutates the RoleBinding to ensure it has the valid params
func mutateBinding(binding *rbacv1.RoleBinding, p bindingParams) {
	binding.Name = generateManagedResourceName(p.vmc.Name)
	binding.Namespace = p.vmc.Namespace
	binding.Labels = p.vmc.Labels

	binding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     p.roleName,
	}
	binding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      p.serviceAccountName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
}

// Generate the common name used by all resources specific to a given managed cluster
func generateManagedResourceName(clusterName string) string {
	return fmt.Sprintf("verrazzano-cluster-%s", clusterName)
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoManagedCluster{}).
		Complete(r)
}

// reconcileManagedClusterDelete performs all necessary cleanup during cluster deletion
func (r *VerrazzanoManagedClusterReconciler) reconcileManagedClusterDelete(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	return r.deleteClusterPrometheusConfiguration(ctx, vmc)
}

func (r *VerrazzanoManagedClusterReconciler) updateStatusNotReady(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string) error {
	now := metav1.Now()
	return r.updateStatus(ctx, vmc, clustersv1alpha1.Condition{Status: corev1.ConditionFalse, Type: clustersv1alpha1.ConditionReady, Message: msg, LastTransitionTime: &now})
}

func (r *VerrazzanoManagedClusterReconciler) updateStatusReady(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string) error {
	now := metav1.Now()
	return r.updateStatus(ctx, vmc, clustersv1alpha1.Condition{Status: corev1.ConditionTrue, Type: clustersv1alpha1.ConditionReady, Message: msg, LastTransitionTime: &now})
}

func (r *VerrazzanoManagedClusterReconciler) handleError(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string, err error, log vzlog.VerrazzanoLogger) {
	fullMsg := fmt.Sprintf("%s: %v", msg, err)
	log.ErrorfThrottled(fullMsg)
	statusErr := r.updateStatusNotReady(ctx, vmc, fullMsg)
	if statusErr != nil {
		log.Errorf("Failed to update status for VMC %s: %v", vmc.Name, statusErr)
	}
}

func (r *VerrazzanoManagedClusterReconciler) updateStatus(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, condition clustersv1alpha1.Condition) error {
	var matchingCondition *clustersv1alpha1.Condition
	var conditionExists bool
	r.log.Debugf("Entered updateStatus for VMC %s, existing conditions = %v", vmc.Name, vmc.Status.Conditions)
	for i, existingCondition := range vmc.Status.Conditions {
		if condition.Type == existingCondition.Type &&
			condition.Status == existingCondition.Status &&
			condition.Message == existingCondition.Message {
			// the exact same condition already exists, don't update
			conditionExists = true
			break
		}
		if condition.Type == existingCondition.Type {
			// use the index here since "existingCondition" is a copy and won't point to the object in the slice
			matchingCondition = &vmc.Status.Conditions[i]
			break
		}
	}
	if !conditionExists {

		if matchingCondition == nil {
			vmc.Status.Conditions = append(vmc.Status.Conditions, condition)
		} else {
			matchingCondition.Message = condition.Message
			matchingCondition.Status = condition.Status
			matchingCondition.LastTransitionTime = condition.LastTransitionTime
		}
	}

	if vmc.Status.LastAgentConnectTime != nil {

		currentTime := metav1.Now()
		//Using the current plus added time to find the difference with lastAgentConnectTime to validate
		//if it exceeds the max allowed time before changing the state of the vmc resource.
		maxPollingTime := currentTime.Add(vzconstants.VMCAgentPollingTimeInterval * vzconstants.MaxTimesVMCAgentPollingTime)
		timeDiff := maxPollingTime.Sub(vmc.Status.LastAgentConnectTime.Time)
		if int(timeDiff.Minutes()) > vzconstants.MaxTimesVMCAgentPollingTime {
			vmc.Status.State = clustersv1alpha1.StateInactive
		} else if vmc.Status.State == "" {
			vmc.Status.State = clustersv1alpha1.StatePending
		} else {
			vmc.Status.State = clustersv1alpha1.StateActive
		}
	}
	r.log.Debugf("Updating Status of VMC %s with condition type %s = %s: %v", vmc.Name, condition.Type, condition.Status, vmc.Status.Conditions)
	return r.Status().Update(ctx, vmc)
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}
