// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	goerrors "errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
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

var (
	reconcileTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "vz_cluster_operator_reconcile_vmc_duration_seconds",
		Help: "The duration of the reconcile process for cluster objects",
	})
	reconcileErrorCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vz_cluster_operator_reconcile_vmc_error_total",
		Help: "The amount of errors encountered in the reconcile process",
	})
	reconcileSuccessCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vz_cluster_operator_reconcile_vmc_success_total",
		Help: "The number of times the reconcile process succeeded",
	})
)

func (r *VerrazzanoManagedClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Time the reconcile process and set the metric with the elapsed time
	startTime := time.Now()
	defer reconcileTimeMetric.Set(time.Since(startTime).Seconds())

	if ctx == nil {
		reconcileErrorCount.Inc()
		return ctrl.Result{}, goerrors.New("context cannot be nil")
	}
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	if err := r.Get(context.TODO(), req.NamespacedName, cr); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			reconcileSuccessCount.Inc()
			return reconcile.Result{}, nil
		}
		reconcileErrorCount.Inc()
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
		reconcileErrorCount.Inc()
		zap.S().Errorf("Failed to create controller logger for VerrazzanoManagedCluster controller", err)
	}

	r.log = log
	log.Oncef("Reconciling Verrazzano resource %v", req.NamespacedName)
	res, err := r.doReconcile(ctx, log, cr)
	if vzctrl.ShouldRequeue(res) {
		reconcileSuccessCount.Inc()
		return res, nil
	}

	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		reconcileErrorCount.Inc()
		return newRequeueWithDelay(), nil
	}

	// The resource has been reconciled.
	log.Oncef("Successfully reconciled VerrazzanoManagedCluster resource %v", req.NamespacedName)

	reconcileSuccessCount.Inc()
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
	vzVMCWaitingForClusterID, err := r.syncManifestSecret(ctx, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the Manifest secret", err, log)
		return newRequeueWithDelay(), err
	}
	if vzVMCWaitingForClusterID {
		// waiting for the cluster ID to be set in the status, so requeue and try again
		return newRequeueWithDelay(), nil
	}

	// create/update a secret with the CA cert from the managed cluster (if any errors occur we just log and continue)
	syncedCert, err := r.syncCACertSecret(vmc)
	if err != nil {
		msg := fmt.Sprintf("Unable to get CA cert from managed cluster %s with id %s: %v", vmc.Name, vmc.Status.RancherRegistration.ClusterID, err)
		r.log.Infof(msg)
		r.setStatusConditionManagedCARetrieved(vmc, corev1.ConditionFalse, msg)
	} else {
		if syncedCert {
			r.setStatusConditionManagedCARetrieved(vmc, corev1.ConditionTrue, "Managed cluster CA cert retrieved successfully")
		}
	}

	log.Debugf("Updating Rancher ClusterRoleBindingTemplate for VMC %s", vmc.Name)
	err = r.updateRancherClusterRoleBindingTemplate(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to update Rancher ClusterRoleBindingTemplate", err, log)
		return newRequeueWithDelay(), err
	}

	log.Debugf("Pushing the Manifest objects for VMC %s", vmc.Name)
	pushedManifest, err := r.pushManifestObjects(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to push the Manifest objects", err, log)
		r.setStatusConditionManifestPushed(vmc, corev1.ConditionFalse, fmt.Sprintf("Failed to push the manifest objects to the managed cluster: %v", err))
		return newRequeueWithDelay(), err
	}
	if pushedManifest {
		r.log.Info("Manifest objects have been successfully pushed to the managed cluster")
		r.setStatusConditionManifestPushed(vmc, corev1.ConditionTrue, "Manifest objects pushed to the managed cluster")
	}

	r.setStatusConditionReady(vmc, "Ready")
	statusErr := r.updateStatus(ctx, vmc)
	if statusErr != nil {
		log.Errorf("Failed to update status to ready for VMC %s: %v", vmc.Name, err)
	}

	if vmc.Status.PrometheusHost == "" {
		log.Infof("Managed cluster Prometheus Host not found in VMC Status for VMC %s. Waiting for VMC to be registered...", vmc.Name)
	} else {
		log.Debugf("Syncing the prometheus scraper for VMC %s", vmc.Name)
		err = r.syncPrometheusScraper(ctx, vmc)
		if err != nil {
			r.handleError(ctx, vmc, "Failed to setup the prometheus scraper for managed cluster", err, log)
			return newRequeueWithDelay(), err
		}
	}

	log.Debugf("Creating or updating keycloak client for %s", vmc.Name)
	err = r.createManagedClusterKeycloakClient(vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to create or update Keycloak client for managed cluster", err, log)
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{Requeue: true, RequeueAfter: constants.ReconcileLoopRequeueInterval}, nil
}

func (r *VerrazzanoManagedClusterReconciler) syncServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Create or update the service account
	_, serviceAccount, err := r.createOrUpdateServiceAccount(context.TODO(), vmc)
	if err != nil {
		return err
	}

	if len(serviceAccount.Secrets) == 0 {
		_, err = r.createServiceAccountTokenSecret(context.TODO(), serviceAccount)
		if err != nil {
			return err
		}
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
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateServiceAccount(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (controllerutil.OperationResult, *corev1.ServiceAccount, error) {
	var serviceAccount corev1.ServiceAccount
	serviceAccount.Namespace = vmc.Namespace
	serviceAccount.Name = generateManagedResourceName(vmc.Name)

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, &serviceAccount, func() error {
		r.mutateServiceAccount(vmc, &serviceAccount)
		// This SetControllerReference call will trigger garbage collection i.e. the serviceAccount
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &serviceAccount, r.Scheme)
	})
	return operationResult, &serviceAccount, err
}

func (r *VerrazzanoManagedClusterReconciler) mutateServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster, serviceAccount *corev1.ServiceAccount) {
	serviceAccount.Name = generateManagedResourceName(vmc.Name)
}

func (r *VerrazzanoManagedClusterReconciler) createServiceAccountTokenSecret(ctx context.Context, serviceAccount *corev1.ServiceAccount) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Name = serviceAccount.Name + "-token"
	secret.Namespace = serviceAccount.Namespace
	secret.Type = corev1.SecretTypeServiceAccountToken
	secret.Annotations = map[string]string{
		corev1.ServiceAccountNameKey: serviceAccount.Name,
	}

	return controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		// This SetControllerReference call will trigger garbage collection i.e. the token secret
		// will automatically get deleted when the service account is deleted
		return controllerutil.SetControllerReference(serviceAccount, &secret, r.Scheme)
	})
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
	if err := r.deleteClusterPrometheusConfiguration(ctx, vmc); err != nil {
		return err
	}
	return r.deleteClusterFromRancher(ctx, vmc)
}

// deleteClusterFromRancher calls the Rancher API to delete the cluster associated with the VMC if the VMC has a cluster id set in the status.
func (r *VerrazzanoManagedClusterReconciler) deleteClusterFromRancher(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if clusterID == "" {
		r.log.Debugf("VMC %s/%s has no Rancher cluster id, skipping delete", vmc.Namespace, vmc.Name)
		return nil
	}

	rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.log)
	if err != nil {
		msg := "Failed to create Rancher API client"
		r.updateRancherStatus(ctx, vmc, clustersv1alpha1.DeleteFailed, clusterID, msg)
		r.log.Errorf("Unable to connect to Rancher API on admin cluster while attempting delete operation: %v", err)
		return err
	}
	if _, err = DeleteClusterFromRancher(rc, clusterID, r.log); err != nil {
		msg := "Failed deleting cluster"
		r.updateRancherStatus(ctx, vmc, clustersv1alpha1.DeleteFailed, clusterID, msg)
		r.log.Errorf("Unable to delete Rancher cluster %s/%s: %v", vmc.Namespace, vmc.Name, err)
		return err
	}

	return nil
}

func (r *VerrazzanoManagedClusterReconciler) setStatusConditionManagedCARetrieved(vmc *clustersv1alpha1.VerrazzanoManagedCluster, value corev1.ConditionStatus, msg string) {
	now := metav1.Now()
	r.setStatusCondition(vmc, clustersv1alpha1.Condition{Status: value, Type: clustersv1alpha1.ConditionManagedCARetrieved, Message: msg, LastTransitionTime: &now}, false)
}

func (r *VerrazzanoManagedClusterReconciler) setStatusConditionManifestPushed(vmc *clustersv1alpha1.VerrazzanoManagedCluster, value corev1.ConditionStatus, msg string) {
	now := metav1.Now()
	r.setStatusCondition(vmc, clustersv1alpha1.Condition{Status: value, Type: clustersv1alpha1.ConditionManifestPushed, Message: msg, LastTransitionTime: &now}, true)
}

// setStatusConditionNotReady sets the status condition Ready = false on the VMC in memory - does NOT update the status in the cluster
func (r *VerrazzanoManagedClusterReconciler) setStatusConditionNotReady(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string) {
	now := metav1.Now()
	r.setStatusCondition(vmc, clustersv1alpha1.Condition{Status: corev1.ConditionFalse, Type: clustersv1alpha1.ConditionReady, Message: msg, LastTransitionTime: &now}, false)
}

// setStatusConditionReady sets the status condition Ready = true on the VMC in memory - does NOT update the status in the cluster
func (r *VerrazzanoManagedClusterReconciler) setStatusConditionReady(vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string) {
	now := metav1.Now()
	r.setStatusCondition(vmc, clustersv1alpha1.Condition{Status: corev1.ConditionTrue, Type: clustersv1alpha1.ConditionReady, Message: msg, LastTransitionTime: &now}, false)
}

func (r *VerrazzanoManagedClusterReconciler) handleError(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, msg string, err error, log vzlog.VerrazzanoLogger) {
	fullMsg := fmt.Sprintf("%s: %v", msg, err)
	log.ErrorfThrottled(fullMsg)
	r.setStatusConditionNotReady(ctx, vmc, fullMsg)
	statusErr := r.updateStatus(ctx, vmc)
	if statusErr != nil {
		log.ErrorfThrottled("Failed to update status for VMC %s: %v", vmc.Name, statusErr)
	}
}

// setStatusCondition updates the VMC status conditions based and replaces already created status conditions
// the onTime flag updates the status condition if the time has changed
func (r *VerrazzanoManagedClusterReconciler) setStatusCondition(vmc *clustersv1alpha1.VerrazzanoManagedCluster, condition clustersv1alpha1.Condition, onTime bool) {
	r.log.Debugf("Entered setStatusCondition for VMC %s for condition %s = %s, existing conditions = %v",
		vmc.Name, condition.Type, condition.Status, vmc.Status.Conditions)
	var matchingCondition *clustersv1alpha1.Condition
	var conditionExists bool
	for i, existingCondition := range vmc.Status.Conditions {
		if condition.Type == existingCondition.Type &&
			condition.Status == existingCondition.Status &&
			condition.Message == existingCondition.Message &&
			(!onTime || condition.LastTransitionTime == existingCondition.LastTransitionTime) {
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
}

// updateStatus updates the status of the VMC in the cluster, with all provided conditions, after setting the vmc.Status.State field for the cluster
func (r *VerrazzanoManagedClusterReconciler) updateStatus(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	if vmc.Status.LastAgentConnectTime != nil {
		currentTime := metav1.Now()
		// Using the current plus added time to find the difference with lastAgentConnectTime to validate
		// if it exceeds the max allowed time before changing the state of the vmc resource.
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

	// Fetch the existing VMC to avoid conflicts in the status update
	existingVMC := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
	if err != nil {
		return err
	}

	// Replace the existing status conditions and state with the conditions generated from this reconcile
	for _, genCondition := range vmc.Status.Conditions {
		r.setStatusCondition(existingVMC, genCondition, genCondition.Type == clustersv1alpha1.ConditionManifestPushed)
	}
	existingVMC.Status.State = vmc.Status.State

	r.log.Debugf("Updating Status of VMC %s: %v", vmc.Name, vmc.Status.Conditions)
	return r.Status().Update(ctx, existingVMC)
}

// getVerrazzanoResource gets the installed Verrazzano resource in the cluster (of which only one is expected)
func (r *VerrazzanoManagedClusterReconciler) getVerrazzanoResource() (*v1beta1.Verrazzano, error) {
	// Get the Verrazzano resource
	verrazzano := v1beta1.VerrazzanoList{}
	err := r.Client.List(context.TODO(), &verrazzano, &client.ListOptions{})
	if err != nil || len(verrazzano.Items) == 0 {
		return nil, fmt.Errorf("Verrazzano must be installed: %v", err)
	}
	return &verrazzano.Items[0], nil
}

func (r *VerrazzanoManagedClusterReconciler) createManagedClusterKeycloakClient(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// login to keycloak
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}

	// create a context that can be leveraged by keycloak method
	ctx, err := spi.NewMinimalContext(r.Client, r.log)
	if err != nil {
		return err
	}

	err = keycloak.LoginKeycloak(ctx, cfg, cli)
	if err != nil {
		return err
	}

	clientId := fmt.Sprintf("verrazzano-%s", vmc.Name)
	err = keycloak.CreateOrUpdateClient(ctx, cfg, cli, clientId, keycloak.ManagedClusterClientTmpl, keycloak.ManagedClusterClientUrisTemplate, false)
	if err != nil {
		return err
	}

	return nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}
