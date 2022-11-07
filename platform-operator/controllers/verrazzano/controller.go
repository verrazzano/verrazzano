// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"bufio"
	"bytes"
	"context"
	goerrors "errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"io"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"strings"
	"sync"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	"github.com/verrazzano/verrazzano/platform-operator/metricsexporter"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Reconciler reconciles a Verrazzano object
type Reconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Controller        controller.Controller
	DryRun            bool
	WatchedComponents map[string]bool
	WatchMutex        *sync.RWMutex
	Bom               *bom.Bom
	StatusUpdater     vzstatus.Updater
}

// Name of finalizer
const finalizerName = "install.verrazzano.io"

// initializedSet is needed to keep track of which Verrazzano CRs have been initialized
var initializedSet = make(map[string]bool)

// systemNamespaceLabels the verrazzano-system namespace labels required
var systemNamespaceLabels = map[string]string{
	"istio-injection":         "enabled",
	"verrazzano.io/namespace": vzconst.VerrazzanoSystemNamespace,
}

// Set to true during unit testing
var unitTesting bool

// Reconcile the Verrazzano CR
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;watch;list;create;update;delete
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		return ctrl.Result{}, goerrors.New("context cannot be nil")
	}
	// Get the Verrazzano resource
	zapLogForMetrics := zap.S().With(log.FieldController, "verrazzano")
	counterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileCounter)
	if err != nil {
		zapLogForMetrics.Error(err)
		return ctrl.Result{}, err
	}
	counterMetricObject.Inc()
	errorCounterMetricObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.ReconcileError)
	if err != nil {
		zapLogForMetrics.Error(err)
		return ctrl.Result{}, err
	}

	reconcileDurationMetricObject, err := metricsexporter.GetDurationMetric(metricsexporter.ReconcileDuration)
	if err != nil {
		zapLogForMetrics.Error(err)
		return ctrl.Result{}, err
	}
	reconcileDurationMetricObject.TimerStart()
	defer reconcileDurationMetricObject.TimerStop()
	vz := &installv1alpha1.Verrazzano{}
	if err := r.Get(ctx, req.NamespacedName, vz); err != nil {
		errorCounterMetricObject.Inc()
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           vz.Name,
		Namespace:      vz.Namespace,
		ID:             string(vz.UID),
		Generation:     vz.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		errorCounterMetricObject.Inc()
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	log.Oncef("Reconciling Verrazzano resource %v, generation %v, version %s", req.NamespacedName, vz.Generation, vz.Status.Version)
	res, err := r.doReconcile(ctx, log, vz)
	if vzctrl.ShouldRequeue(res) {
		return res, nil
	}

	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		errorCounterMetricObject.Inc()
		return newRequeueWithDelay(), nil
	}
	// The Verrazzano resource has been reconciled.
	log.Oncef("Finished reconciling Verrazzano resource %v", req.NamespacedName)
	metricsexporter.AnalyzeVerrazzanoResourceMetrics(log, *vz)

	return ctrl.Result{}, nil
}

// doReconcile the Verrazzano CR
func (r *Reconciler) doReconcile(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Check if uninstalling
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	// Initialize once for this Verrazzano resource when the operator starts
	result, err := r.initForVzResource(vz, log)
	if err != nil {
		return result, err
	}
	if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// Init the state to Ready if this CR has never been processed
	// Always requeue to update cache, ignore error since requeue anyway
	if len(vz.Status.State) == 0 {
		r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		return reconcile.Result{Requeue: true}, nil
	}

	vzctx, err := vzcontext.NewVerrazzanoContext(log, r.Client, vz, r.DryRun)
	if err != nil {
		log.Errorf("Failed to create component context: %v", err)
		return newRequeueWithDelay(), err
	}

	// Process CR based on state
	switch vz.Status.State {
	case installv1alpha1.VzStateFailed:
		return r.ProcFailedState(vzctx)
	case installv1alpha1.VzStateReconciling:
		return r.ProcReconcilingState(vzctx)
	case installv1alpha1.VzStateReady:
		return r.ProcReadyState(vzctx)
	case installv1alpha1.VzStateUpgrading:
		return r.ProcUpgradingState(vzctx)
	case installv1alpha1.VzStatePaused:
		return r.ProcPausedUpgradeState(vzctx)
	default:
		panic("Invalid Verrazzano controller state")
	}
}

// ProcReadyState processes the CR while in the ready state
func (r *Reconciler) ProcReadyState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	log := vzctx.Log
	actualCR := vzctx.ActualCR

	log.Debugf("Entering ProcReadyState")
	ctx := context.TODO()

	// Pre-populate the component status fields
	result, err := r.initializeComponentStatus(log, actualCR)
	if err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// If Verrazzano is installed see if upgrade is needed
	if isInstalled(actualCR.Status) {
		if len(actualCR.Spec.Version) > 0 {
			specVersion, err := semver.NewSemVersion(actualCR.Spec.Version)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			statusVersion, err := semver.NewSemVersion(actualCR.Status.Version)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			// if the spec version field is set and the SemVer spec field doesn't equal the SemVer status field
			if specVersion.CompareTo(statusVersion) != 0 {
				// Transition to upgrade state
				r.updateVzState(log, actualCR, installv1alpha1.VzStateUpgrading)
				return newRequeueWithDelay(), err
			}
		}

		// Keep retrying to reconcile components until it completes
		if result, err := r.reconcileComponents(vzctx, false); err != nil {
			return newRequeueWithDelay(), err
		} else if vzctrl.ShouldRequeue(result) {
			return result, nil
		}

		// Delete leftover MySQL backup job if we find one.
		err = r.cleanupMysqlBackupJob(log)
		if err != nil {
			return newRequeueWithDelay(), err
		}

		return ctrl.Result{}, nil
	}

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if actualCR.Spec.Components.DNS != nil && actualCR.Spec.Components.DNS.OCI != nil {
		err := r.doesOCIDNSConfigSecretExist(actualCR)
		if err != nil {
			return newRequeueWithDelay(), err
		}
	}

	// Pre-create the Verrazzano System namespace if it doesn't already exist, before kicking off the install job,
	// since it is needed for the subsequent step to syncLocalRegistration secret.
	if err := r.createVerrazzanoSystemNamespace(ctx, actualCR, log); err != nil {
		return newRequeueWithDelay(), err
	}

	// Sync the local cluster registration secret that allows the use of MC xyz resources on the
	// admin cluster without needing a VMC.
	if err := r.syncLocalRegistrationSecret(); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return newRequeueWithDelay(), err
	}

	// Change the state back to ready if install complete otherwise requeue
	done, err := r.checkInstallComplete(vzctx)
	if err != nil {
		return newRequeueWithDelay(), err
	}
	if done {
		return ctrl.Result{}, nil
	}

	// Delete leftover uninstall job if we find one.
	err = r.cleanupUninstallJob(buildUninstallJobName(actualCR.Name), getInstallNamespace(), log)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Change the state to installing
	err = r.setInstallingState(log, actualCR)
	if err != nil {
		log.ErrorfThrottled("Error writing Install Started condition to the Verrazzano status: %v", err)
	}
	return newRequeueWithDelay(), err
}

// ProcReconcilingState processes the CR while in the installing state
func (r *Reconciler) ProcReconcilingState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	log := vzctx.Log
	log.Debug("Entering ProcReconcilingState")

	if result, err := r.reconcileComponents(vzctx, false); err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// Change the state back to ready if install complete otherwise requeue
	done, err := r.checkInstallComplete(vzctx)
	if !done || err != nil {
		return newRequeueWithDelay(), err
	}
	log.Once("Successfully installed Verrazzano")
	return ctrl.Result{}, nil
}

// ProcUpgradingState processes the CR while in the upgrading state
func (r *Reconciler) ProcUpgradingState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	actualCR := vzctx.ActualCR
	log := vzctx.Log
	log.Debug("Entering ProcUpgradingState")

	// check for need to pause the upgrade due to VPO update
	if bomVersion, isNewer := isOperatorNewerVersionThanCR(actualCR.Spec.Version); isNewer {
		// upgrade needs to be restarted due to newer operator
		log.Progressf("Upgrade is being paused pending Verrazzano version update to version %s", bomVersion)

		err := r.updateStatus(log, actualCR,
			fmt.Sprintf("Verrazzano upgrade to version %s paused. Upgrade will be performed when version is updated to %s", actualCR.Spec.Version, bomVersion),
			installv1alpha1.CondUpgradePaused, nil)
		return newRequeueWithDelay(), err
	}

	// Install certain components pre-upgrade, like network policies
	if result, err := r.reconcileComponents(vzctx, true); err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// Only upgrade if Version has changed.  When upgrade completes, it will update the status version, see upgrade.go
	if len(actualCR.Spec.Version) > 0 && actualCR.Spec.Version != actualCR.Status.Version {
		if result, err := r.reconcileUpgrade(log, actualCR); err != nil {
			return newRequeueWithDelay(), err
		} else if vzctrl.ShouldRequeue(result) {
			return result, nil
		}
	}

	// Install components that should be installed after upgrade
	if result, err := r.reconcileComponents(vzctx, false); err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	if done, err := r.checkUpgradeComplete(vzctx); !done || err != nil {
		log.Progressf("Upgrade is waiting for all components to enter a Ready state before completion")
		return newRequeueWithDelay(), err
	}

	// Upgrade done along with any post-upgrade installations of new components that are enabled by default.
	msg := fmt.Sprintf("Verrazzano successfully upgraded to version %s", actualCR.Spec.Version)
	log.Once(msg)
	return ctrl.Result{}, nil
}

// ProcPausedUpgradeState processes the CR while in the paused upgrade state
func (r *Reconciler) ProcPausedUpgradeState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	vz := vzctx.ActualCR
	log := vzctx.Log
	log.Debug("Entering ProcPausedUpgradeState")

	// Check if Verrazzano resource is being deleted
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(context.TODO(), log, vz)
	}

	// check if the VPO and VZ versions are the same and the upgrade can proceed
	if isOperatorSameVersionAsCR(vz.Spec.Version) {
		// upgrade can proceed from paused state
		log.Debugf("Restarting upgrade since VZ version and VPO version match")
		r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		// requeue for a fairly long time considering this may be a terminating VPO
		return newRequeueWithDelay(), nil
	}

	return newRequeueWithDelay(), nil
}

// ProcFailedState only allows uninstall
func (r *Reconciler) ProcFailedState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	vz := vzctx.ActualCR
	log := vzctx.Log
	log.Debug("Entering ProcFailedState")
	ctx := context.TODO()

	// Update uninstall status
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	// Determine if the user specified to retry upgrade
	retry, err := r.retryUpgrade(ctx, vz)
	if err != nil {
		log.Errorf("Failed to update the annotations: %v", err)
		return newRequeueWithDelay(), err
	}

	if retry {
		// Log the retry and set the CompStateType to ready, then requeue
		log.Debugf("Restart Version annotation has changed, retrying upgrade")
		r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, nil
	}

	// if annotations didn't trigger a retry, see if a newer version of BOM should
	if bomVersion, isNewer := isOperatorNewerVersionThanCR(vz.Spec.Version); isNewer {
		// upgrade needs to be restarted due to newer operator
		log.Progressf("Upgrade is being paused pending Verrazzano version update to version %s", bomVersion)

		err := r.updateStatus(log, vz,
			fmt.Sprintf("Verrazzano upgrade to version %s paused. Upgrade will be performed when version is updated to %s", vz.Spec.Version, bomVersion),
			installv1alpha1.CondUpgradePaused, nil)
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{}, nil
}

// doesOCIDNSConfigSecretExist returns true if the DNS secret exists
func (r *Reconciler) doesOCIDNSConfigSecretExist(vz *installv1alpha1.Verrazzano) error {
	// ensure the secret exists before proceeding
	secret := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: vz.Spec.Components.DNS.OCI.OCIConfigSecret, Namespace: vzconst.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		return err
	}
	return nil
}

// deleteServiceAccount deletes the service account used for install
func (r *Reconciler) deleteServiceAccount(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano, namespace string) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      buildServiceAccountName(vz.Name),
		},
	}
	err := r.Delete(ctx, &sa, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting ServiceAccount %s: %v", sa.Name, err)
		return err
	}
	return nil
}

// deleteClusterRoleBinding deletes the cluster role binding
func (r *Reconciler) deleteClusterRoleBinding(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildClusterRoleBindingName(vz.Namespace, vz.Name),
		},
	}
	err := r.Delete(ctx, binding, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting ClusterRoleBinding %s: %v", binding.Name, err)
		return err
	}
	return nil
}

// cleanupUninstallJob checks for the existence of a stale uninstall job and deletes the job if one is found
func (r *Reconciler) cleanupUninstallJob(jobName string, namespace string, log vzlog.VerrazzanoLogger) error {
	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Debugf("Checking if stale uninstall job %s exists", jobName)
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: namespace}, jobFound)
	if err == nil {
		log.Debugf("Deleting stale uninstall job %s", jobName)
		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
		err = r.Delete(context.TODO(), jobFound, deleteOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

// cleanupMysqlBackupJob checks for the existence of a stale MySQL restore job and deletes the job if one is found
func (r *Reconciler) cleanupMysqlBackupJob(log vzlog.VerrazzanoLogger) error {
	// Check if jobs for running the restore jobs exist
	jobsFound := &batchv1.JobList{}
	err := r.List(context.TODO(), jobsFound, &client.ListOptions{Namespace: mysql.ComponentNamespace})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, job := range jobsFound.Items {
		// get and inspect the job pods to see if restore container is completed
		podList := &corev1.PodList{}
		podReq, _ := kblabels.NewRequirement("job-name", selection.Equals, []string{job.Name})
		podLabelSelector := kblabels.NewSelector()
		podLabelSelector = podLabelSelector.Add(*podReq)
		err := r.List(context.TODO(), podList, &client.ListOptions{LabelSelector: podLabelSelector})
		if err != nil {
			return err
		}
		backupJob := job
		for i := range podList.Items {
			jobPod := &podList.Items[i]
			if isJobExecutionContainerCompleted(jobPod) {
				// persist the job logs
				persisted := persistJobLog(backupJob, jobPod, log)
				if !persisted {
					log.Infof("Unable to persist job log for %s", backupJob.Name)
				}
				// can delete job since pod has completed
				log.Debugf("Deleting stale backup job %s", job.Name)
				propagationPolicy := metav1.DeletePropagationBackground
				deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
				err = r.Delete(context.TODO(), &backupJob, deleteOptions)
				if err != nil {
					return err
				}

				return nil
			}

			return fmt.Errorf("Pod %s has not completed the database backup", backupJob.Name)
		}
	}

	return nil
}

// persistJobLog will persist the backup job log to the VPO log
func persistJobLog(backupJob batchv1.Job, jobPod *corev1.Pod, log vzlog.VerrazzanoLogger) bool {
	containerName := mysqloperator.BackupContainerName
	if strings.Contains(backupJob.Name, "-schedule-") {
		containerName = containerName + "-cron"
	}
	podLogOpts := corev1.PodLogOptions{Container: containerName}
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false
	}
	req := clientSet.CoreV1().Pods(jobPod.Namespace).GetLogs(jobPod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return false
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(buf)
	scanner.Split(bufio.ScanLines)
	log.Debugf("---------- Begin backup job %s log ----------", backupJob.Name)
	for scanner.Scan() {
		log.Debug(scanner.Text())
	}
	log.Debugf("---------- End backup job %s log ----------", backupJob.Name)

	return true
}

// isJobExecutionContainerCompleted checks to see whether the backup container has terminated with an exit code of 0
func isJobExecutionContainerCompleted(pod *corev1.Pod) bool {
	for _, container := range pod.Status.ContainerStatuses {
		if strings.HasPrefix(container.Name, mysqloperator.BackupContainerName) && container.State.Terminated != nil && container.State.Terminated.ExitCode == 0 {
			return true
		}
	}
	return false
}

// deleteNamespace deletes a namespace
func (r *Reconciler) deleteNamespace(ctx context.Context, log vzlog.VerrazzanoLogger, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace, // required by the controller Delete call
		},
	}
	err := r.Delete(ctx, &ns, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting namespace %s: %v", ns.Name, err)
		return err
	}
	return nil
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Verrazzano{}).Build(r)
	return err
}

// buildUninstallJobName returns the name of an uninstall job based on Verrazzano resource name.
func buildUninstallJobName(name string) string {
	return fmt.Sprintf("verrazzano-uninstall-%s", name)
}

// buildServiceAccountName returns the service account name for jobs based on Verrazzano resource name.
func buildServiceAccountName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// buildClusterRoleBindingName returns the ClusgterRoleBinding name for jobs based on Verrazzano resource name.
func buildClusterRoleBindingName(namespace string, name string) string {
	return fmt.Sprintf("verrazzano-install-%s-%s", namespace, name)
}

func isOperatorSameVersionAsCR(vzVersion string) bool {
	bomVersion, currentVersion, ok := getVzAndOperatorVersions(vzVersion)
	if ok {
		return bomVersion.CompareTo(currentVersion) == 0
	}
	return false
}

func isOperatorNewerVersionThanCR(vzVersion string) (string, bool) {
	bomVersion, currentVersion, ok := getVzAndOperatorVersions(vzVersion)
	if ok {
		return bomVersion.ToString(), bomVersion.CompareTo(currentVersion) > 0
	}
	return "", false
}

func getVzAndOperatorVersions(vzVersion string) (*semver.SemVersion, *semver.SemVersion, bool) {
	bomVersion, err := validators.GetCurrentBomVersion()
	if err != nil {
		return nil, nil, false
	}
	currentVersion, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return nil, nil, false
	}
	return bomVersion, currentVersion, true
}

func (r *Reconciler) getBOM() (*bom.Bom, error) {
	if r.Bom == nil {
		bom, err := bom.NewBom(config.GetDefaultBOMFilePath())
		if err != nil {
			return nil, err
		}
		r.Bom = &bom
	}
	return r.Bom, nil
}

// createVerrazzanoSystemNamespace creates the Verrazzano system namespace if it does not already exist
func (r *Reconciler) createVerrazzanoSystemNamespace(ctx context.Context, cr *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) error {
	// remove injection label if disabled
	istio := cr.Spec.Components.Istio
	if istio != nil && !istio.IsInjectionEnabled() {
		log.Infof("Disabling istio sidecar injection for Verrazzano system components")
		systemNamespaceLabels["istio-injection"] = "disabled"
	}
	log.Debugf("Verrazzano system namespace labels: %v", systemNamespaceLabels)
	// First check if VZ system namespace exists. If not, create it.
	var vzSystemNS corev1.Namespace
	err := r.Get(ctx, types.NamespacedName{Name: vzconst.VerrazzanoSystemNamespace}, &vzSystemNS)
	if err != nil {
		log.Debugf("Creating Verrazzano system namespace")
		if !errors.IsNotFound(err) {
			log.Errorf("Failed to get namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		vzSystemNS.Name = vzconst.VerrazzanoSystemNamespace
		vzSystemNS.Labels, _ = mergeMaps(nil, systemNamespaceLabels)
		log.Oncef("Creating Verrazzano system namespace. Labels: %v", vzSystemNS.Labels)
		if err := r.Create(ctx, &vzSystemNS); err != nil {
			log.Errorf("Failed to create namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		return nil
	}
	// Namespace exists, see if we need to add the label
	log.Oncef("Updating Verrazzano system namespace")
	var updated bool
	vzSystemNS.Labels, updated = mergeMaps(vzSystemNS.Labels, systemNamespaceLabels)
	if !updated {
		return nil
	}
	if err := r.Update(ctx, &vzSystemNS); err != nil {
		log.Errorf("Failed to update namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
		return err
	}
	return nil
}

// mergeMaps Merge one map into another, creating new one if necessary; returns the updated map and true if it was modified
func mergeMaps(to map[string]string, from map[string]string) (map[string]string, bool) {
	mergedMap := to
	if mergedMap == nil {
		mergedMap = make(map[string]string)
	}
	var updated bool
	for k, v := range from {
		if existingVal, ok := mergedMap[k]; !ok {
			mergedMap[k] = v
			updated = true
		} else {
			// check to see if the value changed and, if it has, treat as an update
			if v != existingVal {
				mergedMap[k] = v
				updated = true
			}
		}
	}
	return mergedMap, updated
}

// Get the install namespace where this controller is running.
func getInstallNamespace() string {
	return vzconst.VerrazzanoInstallNamespace
}

func (r *Reconciler) retryUpgrade(ctx context.Context, vz *installv1alpha1.Verrazzano) (bool, error) {
	// get the user-specified restart version - if it's missing then there's nothing to do here
	restartVersion, ok := vz.Annotations[vzconst.UpgradeRetryVersion]
	if !ok {
		return false, nil
	}

	// get the annotation with the previous restart version - if it's missing or the versions do not
	// match, then return true
	prevRestartVersion, ok := vz.Annotations[vzconst.ObservedUpgradeRetryVersion]
	if !ok || restartVersion != prevRestartVersion {

		// add/update the previous restart version annotation to the CR
		vz.Annotations[vzconst.ObservedUpgradeRetryVersion] = restartVersion
		err := r.Client.Update(ctx, vz)
		return true, err
	}
	return false, nil
}

// Process the Verrazzano resource deletion
func (r *Reconciler) procDelete(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// If finalizer is gone then uninstall is done
	if !vzstring.SliceContainsString(vz.ObjectMeta.Finalizers, finalizerName) {
		return ctrl.Result{}, nil
	}
	log.Once("Deleting Verrazzano installation")

	if err := r.setUninstallCondition(log, vz, installv1alpha1.CondUninstallStarted, "Verrazzano uninstall starting"); err != nil {
		return newRequeueWithDelay(), err
	}

	// Uninstall all components
	log.Oncef("Uninstalling components")
	if result, err := r.reconcileUninstall(log, vz); err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	if err := r.setUninstallCondition(log, vz, installv1alpha1.CondUninstallComplete, "Verrazzano uninstall completed"); err != nil {
		return newRequeueWithDelay(), err
	}

	// All install related resources have been deleted, delete the finalizer so that the Verrazzano
	// resource can get removed from etcd.
	log.Oncef("Removing finalizer %s", finalizerName)
	vz.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(vz.ObjectMeta.Finalizers, finalizerName)
	if err := r.Update(ctx, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	delete(initializedSet, vz.Name)

	// Delete the uninstall tracker so the memory can be freed up
	DeleteUninstallTracker(vz)

	return ctrl.Result{}, nil
}

// cleanupOld deletes the resources that used to be in the default namespace in earlier versions of Verrazzano.  This
// also includes the ClusterRoleBinding, which is outside the scope of namespace
func (r *Reconciler) cleanupOld(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Delete ClusterRoleBinding
	err := r.deleteClusterRoleBinding(ctx, log, vz)
	if err != nil {
		return err
	}

	// Delete install service account
	err = r.deleteServiceAccount(ctx, log, vz, vzconst.DefaultNamespace)
	if err != nil {
		return err
	}

	return nil
}

// Create a new Result that will cause a reconciliation requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}

// watchResources Watches resources associated with this vz resource.  The loop to reconcile will be called
// when a resource event is generated.
func (r *Reconciler) watchResources(namespace string, name string, log vzlog.VerrazzanoLogger) error {
	// Watch pods and trigger reconciles for Verrazzano resources when a pod is created
	log.Debugf("Watching for pods to activate reconcile for Verrazzano CR %s/%s", namespace, name)
	err := r.Controller.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		createReconcileEventHandler(namespace, name),
		createPredicate(func(e event.CreateEvent) bool {
			// Cast object to pod
			pod := e.Object.(*corev1.Pod)

			// Filter events to only be for the MySQL namespace
			if pod.Namespace != mysql.ComponentNamespace {
				return false
			}

			// Do not process the event if the pod restarted is not MySQL
			if !strings.HasPrefix(pod.Name, mysql.ComponentName) {
				return false
			}
			log.Debugf("Pod %s in namespace %s created", pod.Name, pod.Namespace)
			r.AddWatch(keycloak.ComponentJSONName)
			return true
		}))
	if err != nil {
		return err
	}

	log.Debugf("Watching for backup jobs to activate reconcile for Verrazzano CR %s/%s", namespace, name)
	err = r.Controller.Watch(
		&source.Kind{Type: &batchv1.Job{}},
		createReconcileEventHandler(namespace, name),
		createPredicate(func(e event.CreateEvent) bool {
			return r.isMysqlOperatorJob(e, log)
		}))
	if err != nil {
		return err
	}
	log.Debugf("Watching for the registration secret to reconcile Verrzzano CR %s/%s", namespace, name)
	return r.Controller.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		createReconcileEventHandler(namespace, name),
		// Reconcile if there is an event related to the registration secret
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return r.isManagedClusterRegistrationSecret(e.Object)
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return r.isManagedClusterRegistrationSecret(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return r.isManagedClusterRegistrationSecret(e.ObjectNew)
			},
		},
	)
}

func (r *Reconciler) isManagedClusterRegistrationSecret(o client.Object) bool {
	secret := o.(*corev1.Secret)
	if secret.Namespace != vzconst.VerrazzanoSystemNamespace || secret.Name != vzconst.MCRegistrationSecret {
		return false
	}
	r.AddWatch(fluentd.ComponentName)
	return true
}

// isMysqlOperatorJob returns true if the job is spawned directly or indirectly by MySQL operator
func (r *Reconciler) isMysqlOperatorJob(e event.CreateEvent, log vzlog.VerrazzanoLogger) bool {
	// Cast object to job
	job := e.Object.(*batchv1.Job)

	// Filter events to only be for the MySQL namespace
	if job.Namespace != mysql.ComponentNamespace {
		return false
	}

	// see if the job ownerReferences point to a cron job owned by the mysql operato
	for _, owner := range job.GetOwnerReferences() {
		if owner.Kind == "CronJob" {
			// get the cronjob reference
			cronJob := &batchv1.CronJob{}
			err := r.Client.Get(context.TODO(), client.ObjectKey{Name: owner.Name, Namespace: job.Namespace}, cronJob)
			if err != nil {
				log.Errorf("Could not find cronjob %s to ascertain job source", owner.Name)
				return false
			}
			return isResourceCreatedByMysqlOperator(cronJob.Labels, log)
		}
	}

	// see if the job has been directly created by the mysql operator
	return isResourceCreatedByMysqlOperator(job.Labels, log)
}

// isResourceCreatedByMysqlOperator checks whether the created-by label is set to "mysql-operator"
func isResourceCreatedByMysqlOperator(labels map[string]string, log vzlog.VerrazzanoLogger) bool {
	createdBy, ok := labels["app.kubernetes.io/created-by"]
	if !ok || createdBy != constants.MySQLOperator {
		return false
	}
	log.Debug("Resource created by MySQL Operator")
	return true
}

func createReconcileEventHandler(namespace, name string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(a client.Object) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}},
			}
		})
}

func createPredicate(f func(e event.CreateEvent) bool) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: f,
	}
}

// initForVzResource will do initialization for the given Verrazzano resource.
// Clean up old resources from a 1.0 release where jobs, etc were in the default namespace
// Add a watch for each Verrazzano resource
func (r *Reconciler) initForVzResource(vz *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	// Add our finalizer if not already added
	if !vzstring.SliceContainsString(vz.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		vz.ObjectMeta.Finalizers = append(vz.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(context.TODO(), vz); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	if unitTesting {
		return ctrl.Result{}, nil
	}

	// Check if init done for this resource
	_, ok := initializedSet[vz.Name]
	if ok {
		return ctrl.Result{}, nil
	}

	// Cleanup old resources that might be left around when the install used to be done
	// in the default namespace
	if err := r.cleanupOld(context.TODO(), log, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	// Watch resources to react with appropriate actions
	if err := r.watchResources(vz.Namespace, vz.Name, log); err != nil {
		log.Errorf("Failed to set Pod watch for Verrazzano CR %s: %v", vz.Name, err)
		return newRequeueWithDelay(), err
	}

	// Update the map indicating the resource is being watched
	initializedSet[vz.Name] = true
	return ctrl.Result{Requeue: true}, nil
}

// This is needed for unit testing
func initUnitTesing() {
	unitTesting = true
}

// AddWatch adds a component to the watched set
func (r *Reconciler) AddWatch(names ...string) {
	r.WatchMutex.Lock()
	defer r.WatchMutex.Unlock()
	for _, name := range names {
		r.WatchedComponents[name] = true
	}
}

func (r *Reconciler) ClearWatch(name string) {
	r.WatchMutex.Lock()
	defer r.WatchMutex.Unlock()
	delete(r.WatchedComponents, name)
}

// IsWatchedComponent checks if a component is watched or not
func (r *Reconciler) IsWatchedComponent(compName string) bool {
	r.WatchMutex.RLock()
	defer r.WatchMutex.RUnlock()
	return r.WatchedComponents[compName]
}
