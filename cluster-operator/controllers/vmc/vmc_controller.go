// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	goerrors "errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	appsv1 "k8s.io/api/apps/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	Scheme             *runtime.Scheme
	RancherIngressHost string
	log                vzlog.VerrazzanoLogger
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
	if err != nil {
		// Never return an error since it has already been logged and we don't want the
		// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
		reconcileErrorCount.Inc()
		return newRequeueWithDelay(), nil
	}
	if vzctrl.ShouldRequeue(res) {
		reconcileSuccessCount.Inc()
		return res, nil
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

	rancherEnabled, err := r.isRancherEnabled()
	if err != nil {
		return newRequeueWithDelay(), err
	}

	log.Debugf("Syncing the Manifest secret for VMC %s", vmc.Name)
	vzVMCWaitingForClusterID, err := r.syncManifestSecret(ctx, rancherEnabled, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the Manifest secret", err, log)
		return newRequeueWithDelay(), err
	}

	// create/update a secret with the CA cert from the managed cluster (if any errors occur we just log and continue)
	syncedCert, err := r.syncCACertSecret(ctx, vmc, rancherEnabled)
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
	pushedManifest, err := r.pushManifestObjects(ctx, rancherEnabled, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to push the Manifest objects", err, log)
		r.setStatusConditionManifestPushed(vmc, corev1.ConditionFalse, fmt.Sprintf("Failed to push the manifest objects to the managed cluster: %v", err))
		return newRequeueWithDelay(), err
	}
	if pushedManifest {
		r.log.Oncef("Manifest objects have been successfully pushed to the managed cluster")
		r.setStatusConditionManifestPushed(vmc, corev1.ConditionTrue, "Manifest objects pushed to the managed cluster")
	}

	log.Debugf("Registering ArgoCD for VMC %s", vmc.Name)
	var argoCDRegistration *clustersv1alpha1.ArgoCDRegistration
	argoCDEnabled, err := r.isArgoCDEnabled()
	if err != nil {
		return newRequeueWithDelay(), err
	}
	if argoCDEnabled && rancherEnabled {
		argoCDRegistration, err = r.registerManagedClusterWithArgoCD(vmc)
		if err != nil {
			r.handleError(ctx, vmc, "Failed to register managed cluster with Argo CD", err, log)
			return newRequeueWithDelay(), err
		}
		vmc.Status.ArgoCDRegistration = *argoCDRegistration
	}
	if !rancherEnabled && argoCDEnabled {
		now := metav1.Now()
		vmc.Status.ArgoCDRegistration = clustersv1alpha1.ArgoCDRegistration{
			Status:    clustersv1alpha1.RegistrationPendingRancher,
			Timestamp: &now,
			Message:   "Skipping Argo CD cluster registration due to Rancher not installed"}
	}

	if !vzVMCWaitingForClusterID {
		r.setStatusConditionReady(vmc, "Ready")
		statusErr := r.updateStatus(ctx, vmc)

		if statusErr != nil {
			log.Errorf("Failed to update status to ready for VMC %s: %v", vmc.Name, statusErr)
		}
	}

	if err := r.syncManagedMetrics(ctx, log, vmc); err != nil {
		return newRequeueWithDelay(), err
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

// syncMultiClusterCASecret gets the CA secret in the VMC from the managed cluster and populates the CA secret for metrics scraping
func (r *VerrazzanoManagedClusterReconciler) syncMultiClusterCASecret(ctx context.Context, log vzlog.VerrazzanoLogger, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (corev1.Secret, error) {
	var secret corev1.Secret

	// read the configuration secret specified if it exists
	if len(vmc.Spec.CASecret) > 0 {
		secretNsn := types.NamespacedName{
			Namespace: vmc.Namespace,
			Name:      vmc.Spec.CASecret,
		}

		// validate secret if it exists
		if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
			return secret, log.ErrorfNewErr("failed to fetch the managed cluster CA secret %s/%s, %v", vmc.Namespace, vmc.Spec.CASecret, err)
		}
	}
	if err := r.mutateManagedClusterCACertsSecret(ctx, vmc, &secret); err != nil {
		return secret, log.ErrorfNewErr("Failed to sync the managed cluster CA certs for VMC %s: %v", vmc.Name, err)
	}
	return secret, nil
}

// mutateManagedClusterCACertsSecret adds and removes managed cluster CA certs to/from the managed cluster CA certs secret
func (r *VerrazzanoManagedClusterReconciler) mutateManagedClusterCACertsSecret(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster, cacrtSecret *corev1.Secret) error {
	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: constants.VerrazzanoMonitoringNamespace}, ns)
	if errors.IsNotFound(err) {
		r.log.Infof("namespace %s does not exist", constants.VerrazzanoMonitoringNamespace)
		return nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.PromManagedClusterCACertsSecretName,
			Namespace: constants.VerrazzanoMonitoringNamespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		if cacrtSecret != nil && cacrtSecret.Data != nil && len(cacrtSecret.Data["cacrt"]) > 0 {
			secret.Data[getCAKey(vmc)] = cacrtSecret.Data["cacrt"]
		} else {
			delete(secret.Data, getCAKey(vmc))
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// syncManagedMetrics syncs the metrics federation for managed clusters
// There are currently two ways of federating metrics from managed clusters:
// 1. Creating a Scrape config for the managed cluster on the admin cluster Prometheus
// 2. Creating a Store in Thanos so that managed cluster metrics can be accessed by the admin cluster Query
// These scenarios are mutually exclusive and the Thanos Query method takes precedence
// There are two conditions that enable the Thanos query method
//  1. Thanos is enabled on the managed cluster
//     a. This manifests as the ThanosHost field in the VMC being populated
//  2. Thanos is enabled on the managed cluster
//
// If these two conditions are not met, the Prometheus federation will be enabled
func (r *VerrazzanoManagedClusterReconciler) syncManagedMetrics(ctx context.Context, log vzlog.VerrazzanoLogger, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// We need to sync the multicluster CA secret for Prometheus and Thanos
	caSecret, err := r.syncMultiClusterCASecret(ctx, log, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to sync the multicluster CA secret", err, log)
	}

	thanosEnabled, err := r.isThanosEnabled()
	if err != nil {
		r.handleError(ctx, vmc, "Failed to verify if Thanos is enabled", err, log)
		return err
	}
	// If the Thanos multicluster requirements are met, set up the Thanos Query store
	if vmc.Status.ThanosQueryStore != "" && thanosEnabled {
		err = r.syncThanosQuery(ctx, vmc)
		if err != nil {
			r.handleError(ctx, vmc, "Failed to update Thanos Query endpoint managed cluster", err, log)
			return err
		}

		// If we successfully sync the managed cluster Thanos Query store, we should remove the federated Prometheus to avoid duplication
		r.log.Oncef("Thanos Query synced for VMC %s. Removing the Prometheus scraper", vmc.Name)
		err = r.deleteClusterPrometheusConfiguration(ctx, vmc)
		if err != nil {
			r.handleError(ctx, vmc, "Failed to remove the Prometheus scrape config", err, log)
			return err
		}
		return nil
	}

	// If Thanos multicluster is disabled, attempt to delete left over resources
	err = r.syncThanosQueryEndpointDelete(ctx, vmc)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to delete Thanos Query endpoint managed cluster", err, log)
		return err
	}

	// If the Prometheus host is not populated, skip federation and do nothing
	if vmc.Status.PrometheusHost == "" {
		// If reached, the managed cluster metrics are not populated, so we should remove the CA cert from the secret
		err := r.mutateManagedClusterCACertsSecret(ctx, vmc, nil)
		if err != nil {
			r.handleError(ctx, vmc, "Failed to delete the managed cluster CA cert from the secret", err, log)
			return err
		}
		log.Oncef("Managed cluster Prometheus Host not found in VMC Status for VMC %s. Waiting for VMC to be registered...", vmc.Name)
		return nil
	}

	// Sync the Prometheus Scraper if Thanos multicluster is disabled and the host is populated
	log.Debugf("Syncing the prometheus scraper for VMC %s", vmc.Name)
	err = r.syncPrometheusScraper(ctx, vmc, &caSecret)
	if err != nil {
		r.handleError(ctx, vmc, "Failed to setup the prometheus scraper for managed cluster", err, log)
		return err
	}

	return nil
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
	if err := r.unregisterClusterFromArgoCD(ctx, vmc); err != nil {
		return err
	}
	if err := r.syncThanosQueryEndpointDelete(ctx, vmc); err != nil {
		return err
	}
	if err := r.mutateManagedClusterCACertsSecret(ctx, vmc, nil); err != nil {
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

	rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.RancherIngressHost, r.log)
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
	if err := r.updateState(vmc); err != nil {
		return err
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
	existingVMC.Status.ArgoCDRegistration = vmc.Status.ArgoCDRegistration

	r.log.Debugf("Updating Status of VMC %s: %v", vmc.Name, vmc.Status.Conditions)
	return r.Status().Update(ctx, existingVMC)
}

// updateState sets the vmc.Status.State for the given VMC.
// The state field functions differently according to whether this VMC references an underlying ClusterAPI cluster.
func (r *VerrazzanoManagedClusterReconciler) updateState(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// If there is no underlying CAPI cluster, set the state field based on the lastAgentConnectTime
	if vmc.Status.ClusterRef == nil {
		r.updateStateFromLastAgentConnectTime(vmc)
		return nil
	}

	// If there is an underlying CAPI cluster, set the state field according to the phase of the CAPI cluster.
	capiClusterPhase, err := r.getCAPIClusterPhase(vmc.Status.ClusterRef)
	if err != nil {
		return err
	}
	if capiClusterPhase != "" {
		vmc.Status.State = capiClusterPhase
	}
	return nil
}

// updateStateFromLastAgentConnectTime sets the vmc.Status.State according to the lastAgentConnectTime,
// setting possible values of Active, Inactive, or Pending.
func (r *VerrazzanoManagedClusterReconciler) updateStateFromLastAgentConnectTime(vmc *clustersv1alpha1.VerrazzanoManagedCluster) {
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
}

// getCAPIClusterPhase returns the phase reported by the CAPI Cluster CR which is referenced by clusterRef.
func (r *VerrazzanoManagedClusterReconciler) getCAPIClusterPhase(clusterRef *clustersv1alpha1.ClusterReference) (clustersv1alpha1.StateType, error) {
	// Get the CAPI Cluster CR
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	clusterNamespacedName := types.NamespacedName{
		Name:      clusterRef.Name,
		Namespace: clusterRef.Namespace,
	}
	if err := r.Get(context.TODO(), clusterNamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}

	// Get the state
	phase, found, err := unstructured.NestedString(cluster.Object, "status", "phase")
	if !found {
		r.log.Progressf("could not find status.phase field inside cluster %s: %v", clusterNamespacedName, err)
		return "", nil
	}
	if err != nil {
		r.log.Progressf("error while looking for status.phase field for cluster %s: %v", clusterNamespacedName, err)
		return "", nil
	}

	// Validate that the CAPI Phase is a proper StateType for the VMC
	switch state := clustersv1alpha1.StateType(phase); state {
	case clustersv1alpha1.StatePending,
		clustersv1alpha1.StateProvisioning,
		clustersv1alpha1.StateProvisioned,
		clustersv1alpha1.StateDeleting,
		clustersv1alpha1.StateUnknown,
		clustersv1alpha1.StateFailed:
		return state, nil
	default:
		r.log.Progressf("retrieved an invalid ClusterAPI Cluster phase of %s", state)
		return clustersv1alpha1.StateUnknown, nil
	}
}

// getVerrazzanoResource gets the installed Verrazzano resource in the cluster (of which only one is expected)
func (r *VerrazzanoManagedClusterReconciler) getVerrazzanoResource() (*v1beta1.Verrazzano, error) {
	// Get the Verrazzano resource
	verrazzano := v1beta1.VerrazzanoList{}
	err := r.Client.List(context.TODO(), &verrazzano, &client.ListOptions{})
	if err != nil || len(verrazzano.Items) == 0 {
		return nil, r.log.ErrorfNewErr("Verrazzano must be installed: %v", err)

	}
	return &verrazzano.Items[0], nil
}

// leveraged to replace method (unit testing)
var createClient = func(r *VerrazzanoManagedClusterReconciler, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	const prometheusHostPrefix = "prometheus.vmi.system"
	promHost := vmc.Status.PrometheusHost
	// Skip Keycloak client generation if Prometheus isn't present in VMC status
	// MCAgent on the managed cluster will set this if/when it is ready
	if len(promHost) == 0 {
		r.log.Debug("Skipping Prometheus Keycloak client creation: VMC Prometheus not found")
		return nil
	}

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

	dnsSubdomain := promHost[len(prometheusHostPrefix)+1:]
	clientID := fmt.Sprintf("verrazzano-%s", vmc.Name)
	err = keycloak.CreateOrUpdateClient(ctx, cfg, cli, clientID, keycloak.ManagedClusterClientTmpl, keycloak.ManagedClusterClientUrisTemplate, false, &dnsSubdomain)
	if err != nil {
		return err
	}

	return nil
}

// createManagedClusterKeycloakClient creates a Keycloak client for the managed cluster
func (r *VerrazzanoManagedClusterReconciler) createManagedClusterKeycloakClient(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	return createClient(r, vmc)
}

// getClusterClient returns a controller runtime client configured for the workload cluster
func (r *VerrazzanoManagedClusterReconciler) getClusterClient(restConfig *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = rbacv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = netv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)

	return client.New(restConfig, client.Options{Scheme: scheme})
}

// getWorkloadClusterKubeconfig returns a kubeconfig for accessing the workload cluster
func (r *VerrazzanoManagedClusterReconciler) getWorkloadClusterKubeconfig(cluster *unstructured.Unstructured) ([]byte, error) {
	// get the cluster kubeconfig
	kubeconfigSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", cluster.GetName()), Namespace: cluster.GetNamespace()}, kubeconfigSecret)
	if err != nil {
		r.log.Progressf("failed to obtain workload cluster kubeconfig resource. Re-queuing...")
		return nil, err
	}
	kubeconfig, ok := kubeconfigSecret.Data["value"]
	if !ok {
		r.log.Error(err, "failed to read kubeconfig from resource")
		return nil, fmt.Errorf("Unable to read kubeconfig from retrieved cluster resource")
	}

	return kubeconfig, nil
}

func (r *VerrazzanoManagedClusterReconciler) getWorkloadClusterClient(cluster *unstructured.Unstructured) (client.Client, error) {
	// identify whether the workload cluster is using "untrusted" certs
	kubeconfig, err := r.getWorkloadClusterKubeconfig(cluster)
	if err != nil {
		// requeue since we're waiting for cluster
		return nil, err
	}
	// create a workload cluster client
	// create workload cluster client
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		r.log.Progress("Failed getting rest config from workload kubeconfig")
		return nil, err
	}
	workloadClient, err := r.getClusterClient(restConfig)
	if err != nil {
		return nil, err
	}
	return workloadClient, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}

func getClusterResourceName(cluster *unstructured.Unstructured, client client.Client) string {
	// check for existence of a Rancher cluster management resource
	rancherMgmtCluster := &unstructured.Unstructured{}
	rancherMgmtCluster.SetGroupVersionKind(common.GetRancherMgmtAPIGVKForKind("Cluster"))
	err := client.Get(context.TODO(), types.NamespacedName{Name: cluster.GetName(), Namespace: cluster.GetNamespace()}, rancherMgmtCluster)
	if err != nil {
		return cluster.GetName()
	}
	// return the display Name
	return rancherMgmtCluster.UnstructuredContent()["spec"].(map[string]interface{})["displayName"].(string)
}
