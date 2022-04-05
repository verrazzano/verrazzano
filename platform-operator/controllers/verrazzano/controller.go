// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/uninstalljob"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s"
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
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
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
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// Get the Verrazzano resource
	vz := &installv1alpha1.Verrazzano{}
	if err := r.Get(context.TODO(), req.NamespacedName, vz); err != nil {
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
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	log.Oncef("Reconciling Verrazzano resource %v, generation %v, version %s", req.NamespacedName, vz.Generation, vz.Status.Version)
	res, err := r.doReconcile(log, vz)
	if vzctrl.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return newRequeueWithDelay(), nil
	}
	// The Verrazzano resource has been reconciled.
	log.Oncef("Finished reconciling Verrazzano resource %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile the Verrazzano CR
func (r *Reconciler) doReconcile(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	ctx := context.TODO()

	// Initialize once for this Verrazzano resource when the operator starts
	result, err := r.initForVzResource(vz, log)
	if err != nil {
		return result, err
	}
	if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// Ensure the required resources needed for both install and uninstall exist
	// This needs to be done for every state since the initForVzResource might have deleted
	// role binding during old resource cleanup.
	if err := r.createServiceAccount(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	}
	if err := r.createClusterRoleBinding(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	// Init the state to Ready if this CR has never been processed
	// Always requeue to update cache, ignore error since requeue anyway
	if len(vz.Status.State) == 0 {
		r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		return reconcile.Result{Requeue: true}, nil
	}

	vzctx, err := vzcontext.NewVerrazzanoContext(log, r, vz, r.DryRun)
	if err != nil {
		log.Errorf("Failed to create component context: %v", err)
		return newRequeueWithDelay(), err
	}

	// Process CR based on state
	switch vz.Status.State {
	case installv1alpha1.VzStateFailed:
		return r.ProcFailedState(vzctx)
	case installv1alpha1.VzStateInstalling:
		return r.ProcInstallingState(vzctx)
	case installv1alpha1.VzStateReady:
		return r.ProcReadyState(vzctx)
	case installv1alpha1.VzStateUninstalling:
		return r.ProcUninstallingState(vzctx)
	case installv1alpha1.VzStateUpgrading:
		return r.ProcUpgradingState(vzctx)
	case installv1alpha1.VzStatePaused:
		return r.ProcPausedUpgradeState(vzctx)
	default:
		panic("Invalid Verrazzano contoller state")
	}
}

// ProcReadyState processes the CR while in the ready state
func (r *Reconciler) ProcReadyState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	log := vzctx.Log
	actualCR := vzctx.ActualCR

	log.Debugf("Entering ProcReadyState")
	ctx := context.TODO()

	// Check if Verrazzano resource is being deleted
	if !actualCR.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, actualCR)
	}

	// Pre-populate the component status fields
	result, err := r.initializeComponentStatus(log, actualCR)
	if err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}

	// If Verrazzano is installed see if upgrade is needed
	if isInstalled(actualCR.Status) {
		// If the version is specified and different from the current version of the installation
		// then proceed with upgrade
		if len(actualCR.Spec.Version) > 0 && actualCR.Spec.Version != actualCR.Status.Version {
			result, err := r.reconcileUpgrade(log, actualCR)
			// Keep retrying the upgrade until it completes.
			if err != nil {
				return newRequeueWithDelay(), err
			} else if vzctrl.ShouldRequeue(result) {
				return result, nil
			}
		}
		// Keep retrying to reconcile components until it completes
		if result, err := r.reconcileComponents(vzctx); err != nil {
			return newRequeueWithDelay(), err
		} else if vzctrl.ShouldRequeue(result) {
			return result, nil
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
	if err := r.createVerrazzanoSystemNamespace(ctx, log); err != nil {
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
	return newRequeueWithDelay(), err
}

// ProcInstallingState processes the CR while in the installing state
func (r *Reconciler) ProcInstallingState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	log := vzctx.Log
	actualCR := vzctx.ActualCR
	log.Debug("Entering ProcInstallingState")
	ctx := context.TODO()

	// Check if Verrazzano resource is being deleted
	if !actualCR.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, actualCR)
	}

	if result, err := r.reconcileComponents(vzctx); err != nil {
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

// ProcUninstallingState processes the CR while in the uninstalling state
func (r *Reconciler) ProcUninstallingState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	vz := vzctx.ActualCR
	log := vzctx.Log
	log.Debug("Entering ProcUninstallingState")
	ctx := context.TODO()

	// Update uninstall status
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	return ctrl.Result{}, nil
}

// ProcUpgradingState processes the CR while in the upgrading state
func (r *Reconciler) ProcUpgradingState(vzctx vzcontext.VerrazzanoContext) (ctrl.Result, error) {
	vz := vzctx.ActualCR
	log := vzctx.Log
	log.Debug("Entering ProcUpgradingState")

	// Check if Verrazzano resource is being deleted
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(context.TODO(), log, vz)
	}

	// check for need to pause the upgrade due to VPO update
	if bomVersion, isNewer := isOperatorNewerVersionThanCR(vz.Spec.Version); isNewer {
		// upgrade needs to be restarted due to newer operator
		log.Progressf("Upgrade is being paused pending Verrazzano version update to version %s", bomVersion)

		err := r.updateStatus(log, vz,
			fmt.Sprintf("Verrazzano upgrade to version %s paused. Upgrade will be performed when version is updated to %s", vz.Spec.Version, bomVersion),
			installv1alpha1.CondUpgradePaused)
		return newRequeueWithDelay(), err
	}

	if result, err := r.reconcileUpgrade(log, vz); err != nil {
		return newRequeueWithDelay(), err
	} else if vzctrl.ShouldRequeue(result) {
		return result, nil
	}
	// Upgrade should always requeue to ensure that reconciler runs post upgrade to install
	// components that may have been waiting for upgrade
	return newRequeueWithDelay(), nil
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
		err := r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		// requeue for a fairly long time considering this may be a terminating VPO
		return newRequeueWithDelay(), err
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
		err = r.updateVzState(log, vz, installv1alpha1.VzStateReady)
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	// if annotations didn't trigger a retry, see if a newer version of BOM should
	if bomVersion, isNewer := isOperatorNewerVersionThanCR(vz.Spec.Version); isNewer {
		// upgrade needs to be restarted due to newer operator
		log.Progressf("Upgrade is being paused pending Verrazzano version update to version %s", bomVersion)

		err := r.updateStatus(log, vz,
			fmt.Sprintf("Verrazzano upgrade to version %s paused. Upgrade will be performed when version is updated to %s", vz.Spec.Version, bomVersion),
			installv1alpha1.CondUpgradePaused)
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

// createServiceAccount creates a required service account
func (r *Reconciler) createServiceAccount(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new service account resource
	imagePullSecrets := strings.Split(os.Getenv("IMAGE_PULL_SECRETS"), ",")
	for i := range imagePullSecrets {
		imagePullSecrets[i] = strings.TrimSpace(imagePullSecrets[i])
	}
	serviceAccount := rbac.NewServiceAccount(getInstallNamespace(), buildServiceAccountName(vz.Name), imagePullSecrets, vz.Labels)

	// Check if the service account for running the scripts exist
	serviceAccountFound := &corev1.ServiceAccount{}
	log.Debugf("Checking if install service account %s exist", buildServiceAccountName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildServiceAccountName(vz.Name), Namespace: getInstallNamespace()}, serviceAccountFound)
	if err != nil && errors.IsNotFound(err) {
		log.Debugf("Creating install service account %s", buildServiceAccountName(vz.Name))
		err = r.Create(ctx, serviceAccount)
		if err != nil {
			log.Errorf("Failed to create install service account %s: %v", buildServiceAccountName(vz.Name), err)
			return err
		}
	} else if err != nil {
		log.Errorf("Failed to get install service account %s: %v", buildServiceAccountName(vz.Name), err)
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

// createClusterRoleBinding creates a required cluster role binding
// NOTE: A RoleBinding doesn't work because we get the following error when the install scripts call kubectl get nodes
//   " nodes is forbidden: User "xyz" cannot list resource "nodes" in API group "" at the cluster scope"
func (r *Reconciler) createClusterRoleBinding(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new cluster role binding resource
	binding := rbac.NewClusterRoleBinding(vz, buildClusterRoleBindingName(vz.Namespace, vz.Name), getInstallNamespace(), buildServiceAccountName(vz.Name))

	// Check if the cluster role binding for running the install scripts exist
	bindingFound := &rbacv1.ClusterRoleBinding{}
	log.Debugf("Checking if install cluster role binding %s exist", binding.Name)
	err := r.Get(ctx, types.NamespacedName{Name: binding.Name, Namespace: binding.Namespace}, bindingFound)
	if err != nil && errors.IsNotFound(err) {
		log.Debugf("Creating install cluster role binding %s", binding.Name)
		err = r.Create(ctx, binding)
		if err != nil {
			log.Errorf("Failed to create install cluster role binding %s: %v", binding.Name, err)
			return err
		}
	} else if err != nil {
		log.Errorf("Failed to get install cluster role binding %s: %v", binding.Name, err)
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

// checkInstallComplete checks to see if the install is complete
func (r *Reconciler) checkInstallComplete(vzctx vzcontext.VerrazzanoContext) (bool, error) {
	log := vzctx.Log
	actualCR := vzctx.ActualCR
	ready, err := r.checkComponentReadyState(vzctx)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	// Set install complete IFF all subcomponent status' are "CompStateReady"
	message := "Verrazzano install completed successfully"
	// Status update must be performed on the actual CR read from K8S
	return true, r.updateStatus(log, actualCR, message, installv1alpha1.CondInstallComplete)
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

// deleteNamespace deletes a namespace
func (r *Reconciler) deleteNamespace(ctx context.Context, log vzlog.VerrazzanoLogger, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      namespace, // required by the controller Delete call
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

func (r *Reconciler) createUninstallJob(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new uninstall job resource
	job := uninstalljob.NewJob(
		&uninstalljob.JobConfig{
			JobConfigCommon: k8s.JobConfigCommon{
				JobName:            buildUninstallJobName(vz.Name),
				Namespace:          getInstallNamespace(),
				Labels:             vz.Labels,
				ServiceAccountName: buildServiceAccountName(vz.Name),
				JobImage:           os.Getenv("VZ_INSTALL_IMAGE"),
				DryRun:             r.DryRun,
			},
		},
	)

	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Debugf("Checking if uninstall job %s exist", buildUninstallJobName(vz.Name))
	err := r.Get(context.TODO(), types.NamespacedName{Name: buildUninstallJobName(vz.Name), Namespace: getInstallNamespace()}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating uninstall job %s, dry-run=%v", buildUninstallJobName(vz.Name), r.DryRun)
		err = r.Create(context.TODO(), job)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	log.Progressf("Uninstall job %s is running", buildUninstallJobName(vz.Name))

	err = r.setUninstallCondition(log, jobFound, vz)
	if err != nil {
		return err
	}

	return nil
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

// buildInternalConfigMapName returns the name of the internal configmap associated with an install resource.
func buildInternalConfigMapName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s-internal", name)
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
	bomVersion, err := installv1alpha1.GetCurrentBomVersion()
	if err != nil {
		return nil, nil, false
	}
	currentVersion, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return nil, nil, false
	}
	return bomVersion, currentVersion, true
}

// updateStatus updates the status in the Verrazzano CR
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	cr.Status.Conditions = append(cr.Status.Conditions, condition)

	// Set the state of resource
	cr.Status.State = conditionToVzState(conditionType)
	log.Debugf("Setting Verrazzano resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzState(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, state installv1alpha1.VzStateType) error {
	// Set the state of resource
	cr.Status.State = state
	log.Debugf("Setting Verrazzano state: %v", cr.Status.State)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
}

func (r *Reconciler) updateComponentStatus(compContext spi.ComponentContext, message string, conditionType installv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}

	componentName := compContext.GetComponent()
	cr := compContext.ActualCR()
	log := compContext.Log()

	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}
	componentStatus := cr.Status.Components[componentName]
	if componentStatus == nil {
		componentStatus = &installv1alpha1.ComponentStatusDetails{
			Name: componentName,
		}
		cr.Status.Components[componentName] = componentStatus
	}
	if conditionType == installv1alpha1.CondInstallComplete {
		cr.Status.VerrazzanoInstance = vzinstance.GetInstanceInfo(compContext)
		if componentStatus.ReconcilingGeneration > 0 {
			componentStatus.LastReconciledGeneration = componentStatus.ReconcilingGeneration
			componentStatus.ReconcilingGeneration = 0
		} else {
			componentStatus.LastReconciledGeneration = cr.Generation
		}
	} else {
		if componentStatus.ReconcilingGeneration == 0 {
			componentStatus.ReconcilingGeneration = cr.Generation
		}
	}
	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus, condition)

	// Set the state of resource
	componentStatus.State = checkCondtitionType(conditionType)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
}

func appendConditionIfNecessary(log vzlog.VerrazzanoLogger, compStatus *installv1alpha1.ComponentStatusDetails, newCondition installv1alpha1.Condition) []installv1alpha1.Condition {
	for _, existingCondition := range compStatus.Conditions {
		if existingCondition.Type == newCondition.Type {
			return compStatus.Conditions
		}
	}
	log.Debugf("Adding %s resource newCondition: %v", compStatus.Name, newCondition.Type)
	return append(compStatus.Conditions, newCondition)
}

func checkCondtitionType(currentCondition installv1alpha1.ConditionType) installv1alpha1.CompStateType {
	switch currentCondition {
	case installv1alpha1.CondPreInstall:
		return installv1alpha1.CompStatePreInstalling
	case installv1alpha1.CondInstallStarted:
		return installv1alpha1.CompStateInstalling
	case installv1alpha1.CondUninstallStarted:
		return installv1alpha1.CompStateUninstalling
	case installv1alpha1.CondUpgradeStarted:
		return installv1alpha1.CompStateUpgrading
	case installv1alpha1.CondUpgradePaused:
		return installv1alpha1.CompStateUpgrading
	case installv1alpha1.CondUninstallComplete:
		return installv1alpha1.CompStateReady
	case installv1alpha1.CondInstallFailed, installv1alpha1.CondUpgradeFailed, installv1alpha1.CondUninstallFailed:
		return installv1alpha1.CompStateFailed
	}
	// Return ready for installv1alpha1.CondInstallComplete, installv1alpha1.CondUpgradeComplete
	return installv1alpha1.CompStateReady
}

// Convert a condition to a VZ State
func conditionToVzState(currentCondition installv1alpha1.ConditionType) installv1alpha1.VzStateType {
	switch currentCondition {
	case installv1alpha1.CondInstallStarted:
		return installv1alpha1.VzStateInstalling
	case installv1alpha1.CondUninstallStarted:
		return installv1alpha1.VzStateUninstalling
	case installv1alpha1.CondUpgradeStarted:
		return installv1alpha1.VzStateUpgrading
	case installv1alpha1.CondUpgradePaused:
		return installv1alpha1.VzStatePaused
	case installv1alpha1.CondUninstallComplete:
		return installv1alpha1.VzStateReady
	case installv1alpha1.CondInstallFailed, installv1alpha1.CondUpgradeFailed, installv1alpha1.CondUninstallFailed:
		return installv1alpha1.VzStateFailed
	}
	// Return ready for installv1alpha1.CondInstallComplete, installv1alpha1.CondUpgradeComplete
	return installv1alpha1.VzStateReady
}

// setInstallStartedCondition
func (r *Reconciler) setInstallingState(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := installv1alpha1.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	vz.Status.Version = bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano install in progress", installv1alpha1.CondInstallStarted)
}

// checkComponentReadyState returns true if all component-level status' are "CompStateReady" for enabled components
func (r *Reconciler) checkComponentReadyState(vzctx vzcontext.VerrazzanoContext) (bool, error) {
	cr := vzctx.ActualCR
	if unitTesting {
		for _, compStatus := range cr.Status.Components {
			if compStatus.State != installv1alpha1.CompStateDisabled && compStatus.State != installv1alpha1.CompStateReady {
				return false, nil
			}
		}
		return true, nil
	}

	// Return false if any enabled component is not ready
	for _, comp := range registry.GetComponents() {
		spiCtx, err := spi.NewContext(vzctx.Log, r, vzctx.ActualCR, r.DryRun)
		if err != nil {
			spiCtx.Log().Errorf("Failed to create component context: %v", err)
			return false, err
		}
		if comp.IsEnabled(spiCtx.EffectiveCR()) && cr.Status.Components[comp.Name()].State != installv1alpha1.CompStateReady {
			return false, nil
		}
	}
	return true, nil
}

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based install.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(log, r, cr, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	statusUpdated := false
	for _, comp := range registry.GetComponents() {
		if status, ok := cr.Status.Components[comp.Name()]; ok {
			if status.LastReconciledGeneration == 0 {
				status.LastReconciledGeneration = cr.Generation
			}
			// Skip components that have already been processed
			continue
		}
		if comp.IsOperatorInstallSupported() {
			// If the component is installed then mark it as ready
			compContext := newContext.Init(comp.Name()).Operation(vzconst.InitializeOperation)
			lastReconciled := int64(0)
			state := installv1alpha1.CompStateDisabled
			if !unitTesting {
				installed, err := comp.IsInstalled(compContext)
				if err != nil {
					log.Errorf("Failed to determine if component %s is installed: %v", comp.Name(), err)
					return newRequeueWithDelay(), err
				}
				if installed {
					state = installv1alpha1.CompStateReady
					lastReconciled = compContext.ActualCR().Generation
				}
			}
			cr.Status.Components[comp.Name()] = &installv1alpha1.ComponentStatusDetails{
				Name:                     comp.Name(),
				State:                    state,
				LastReconciledGeneration: lastReconciled,
			}
			statusUpdated = true
		}
	}
	// Update the status
	if statusUpdated {
		return newRequeueWithDelay(), r.updateVerrazzanoStatus(log, cr)
	}
	return ctrl.Result{}, nil
}

// setUninstallCondition sets the Verrazzano resource condition in status for uninstall
func (r *Reconciler) setUninstallCondition(log vzlog.VerrazzanoLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
	// If the job has succeeded or failed add the appropriate condition
	if job.Status.Succeeded != 0 || job.Status.Failed != 0 {
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.CondUninstallComplete || condition.Type == installv1alpha1.CondUninstallFailed {
				return nil
			}
		}

		// Remove the owner reference so that the install job is not deleted when the Verrazzano resource is deleted
		job.SetOwnerReferences([]metav1.OwnerReference{})

		// Update the job
		err := r.Status().Update(context.TODO(), job)
		if err != nil {
			log.Errorf("Failed to update uninstall job owner references: %v", err)
			return err
		}

		var message string
		var conditionType installv1alpha1.ConditionType
		if job.Status.Succeeded == 1 {
			message = "Successfullly uninstalled Verrazzano"
			conditionType = installv1alpha1.CondUninstallComplete
			log.Info(message)
		} else {
			message = "Failed to uninstall Verrazzano"
			conditionType = installv1alpha1.CondUninstallFailed
			log.Error(message)
		}
		return r.updateStatus(log, vz, message, conditionType)
	}

	// Add the uninstall started condition if not already added
	for _, condition := range vz.Status.Conditions {
		if condition.Type == installv1alpha1.CondUninstallStarted {
			return nil
		}
	}

	return r.updateStatus(log, vz, "CompStateInstalling Verrazzano", installv1alpha1.CondUninstallStarted)
}

// getInternalConfigMap Convenience method for getting the saved install ConfigMap
func (r *Reconciler) getInternalConfigMap(ctx context.Context, vz *installv1alpha1.Verrazzano) (installConfig *corev1.ConfigMap, err error) {
	key := client.ObjectKey{
		Namespace: getInstallNamespace(),
		Name:      buildInternalConfigMapName(vz.Name),
	}
	installConfig = &corev1.ConfigMap{}
	err = r.Get(ctx, key, installConfig)
	return installConfig, err
}

// createVerrazzanoSystemNamespace creates the Verrazzano system namespace if it does not already exist
func (r *Reconciler) createVerrazzanoSystemNamespace(ctx context.Context, log vzlog.VerrazzanoLogger) error {
	// First check if VZ system namespace exists. If not, create it.
	var vzSystemNS corev1.Namespace
	err := r.Get(ctx, types.NamespacedName{Name: vzconst.VerrazzanoSystemNamespace}, &vzSystemNS)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed to get namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		vzSystemNS.Name = vzconst.VerrazzanoSystemNamespace
		vzSystemNS.Labels, _ = mergeMaps(nil, systemNamespaceLabels)
		if err := r.Create(ctx, &vzSystemNS); err != nil {
			log.Errorf("Failed to create namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		return nil
	}
	// Namespace exists, see if we need to add the label
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
		if _, ok := mergedMap[k]; !ok {
			mergedMap[k] = v
			updated = true
		}
	}
	return mergedMap, updated
}

// buildDomain Build the DNS Domain from the current install
func buildDomain(log vzlog.VerrazzanoLogger, c client.Client, vz *installv1alpha1.Verrazzano) (string, error) {
	subdomain := vz.Spec.EnvironmentName
	if len(subdomain) == 0 {
		subdomain = vzconst.DefaultEnvironmentName
	}
	baseDomain, err := buildDomainSuffix(log, c, vz)
	if err != nil {
		return "", err
	}
	domain := subdomain + "." + baseDomain
	return domain, nil
}

// buildDomainSuffix Get the configured domain suffix, or compute the nip.io domain
func buildDomainSuffix(log vzlog.VerrazzanoLogger, c client.Client, vz *installv1alpha1.Verrazzano) (string, error) {
	dns := vz.Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return dns.OCI.DNSZoneName, nil
	}
	if dns != nil && dns.External != nil {
		return dns.External.Suffix, nil
	}
	ipAddress, err := getIngressIP(log, c)
	if err != nil {
		return "", err
	}

	if dns != nil && dns.Wildcard != nil {
		return ipAddress + dns.Wildcard.Domain, nil
	}

	// Default to nip.io
	return ipAddress + ".nip.io", nil
}

// getIngressIP get the Ingress IP, used for the wildcard case (magic DNS)
func getIngressIP(log vzlog.VerrazzanoLogger, c client.Client) (string, error) {
	const nginxIngressController = "ingress-controller-ingress-nginx-controller"
	const nginxNamespace = "ingress-nginx"
	nsn := types.NamespacedName{Name: nginxIngressController, Namespace: nginxNamespace}
	nginxService := corev1.Service{}
	err := c.Get(context.TODO(), nsn, &nginxService)
	if err != nil {
		log.Errorf("Failed to get service %v: %v", nsn, err)
		return "", err
	}
	if nginxService.Spec.Type == corev1.ServiceTypeLoadBalancer || nginxService.Spec.Type == corev1.ServiceTypeNodePort {
		nginxIngress := nginxService.Status.LoadBalancer.Ingress
		if len(nginxIngress) == 0 {
			// In case of OLCNE, need to obtain the External IP from the Spec
			if len(nginxService.Spec.ExternalIPs) == 0 {
				return "", log.ErrorfNewErr("Failed because NGINX service %s is missing External IP address", nginxService.Name)
			}
			return nginxService.Spec.ExternalIPs[0], nil
		}
		return nginxIngress[0].IP, nil
	}
	err = fmt.Errorf("Failed because of unsupported service type %s for NGINX ingress", string(nginxService.Spec.Type))
	log.Errorf("%v", err)
	return "", err
}

func addFluentdExtraVolumeMounts(files []string, vz *installv1alpha1.Verrazzano) *installv1alpha1.Verrazzano {
	for _, extraMount := range dirsOutsideVarLog(files) {
		if vz.Spec.Components.Fluentd == nil {
			vz.Spec.Components.Fluentd = &installv1alpha1.FluentdComponent{}
		}
		found := false
		for _, vm := range vz.Spec.Components.Fluentd.ExtraVolumeMounts {
			if isParentDir(extraMount, vm.Source) {
				found = true
			}
		}
		if !found {
			vz.Spec.Components.Fluentd.ExtraVolumeMounts = append(vz.Spec.Components.Fluentd.ExtraVolumeMounts,
				installv1alpha1.VolumeMount{Source: extraMount})
		}
	}
	return vz
}

func dirsOutsideVarLog(paths []string) []string {
	var results []string
	for _, path := range paths {
		if !strings.HasPrefix(path, "/var/log/") {
			found := false
			var temp []string
			for _, res := range results {
				commonPath := commonPath(res, path)
				if commonPath != "/" {
					temp = append(temp, commonPath)
					found = true
				} else {
					temp = append(temp, res)
				}
			}
			if !found {
				temp = append(temp, path)
			}
			results = temp
		}
	}
	return results
}

func isParentDir(path, dir string) bool {
	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	return commonPath(path, dir) == dir
}

func commonPath(a, b string) string {
	i := 0
	s := 0
	for ; i < len(a) && i < len(b) && a[i] == b[i]; i++ {
		if a[i] == '/' {
			s = i
		}
	}
	return a[0 : s+1]
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

	// Create the uninstall job if it doesn't exist
	if err := r.createUninstallJob(log, vz); err != nil {
		if errors.IsConflict(err) {
			log.Debug("Resource conflict creating the uninstall job, requeuing")
		} else {
			log.Errorf("Failed creating the uninstall job: %v", err)
		}
		return newRequeueWithDelay(), err
	}

	// Remove the finalizer and update the Verrazzano resource if the uninstall has finished.
	for _, condition := range vz.Status.Conditions {
		if condition.Type == installv1alpha1.CondUninstallComplete || condition.Type == installv1alpha1.CondUninstallFailed {
			if condition.Type == installv1alpha1.CondUninstallComplete {
				log.Once("Successfully uninstalled Verrrazzano")
			} else {
				log.Once("Failed uninstalling Verraazzano")
			}

			err := r.cleanup(ctx, log, vz)
			if err != nil {
				return newRequeueWithDelay(), err
			}

			// All install related resources have been deleted, delete the finalizer so that the Verrazzano
			// resource can get removed from etcd.
			log.Debugf("Removing finalizer %s", finalizerName)
			vz.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(vz.ObjectMeta.Finalizers, finalizerName)
			err = r.Update(ctx, vz)
			if err != nil {
				return newRequeueWithDelay(), err
			}

			delete(initializedSet, vz.Name)
			// Uninstall is done, all cleanup is finished, and finalizer removed.
			return ctrl.Result{}, nil
		}
	}
	return newRequeueWithDelay(), nil
}

// Cleanup the resources left over from install and uninstall
func (r *Reconciler) cleanup(ctx context.Context, log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Delete roleBinding
	err := r.deleteClusterRoleBinding(ctx, log, vz)
	if err != nil {
		return err
	}

	// Delete install service account
	err = r.deleteServiceAccount(ctx, log, vz, getInstallNamespace())
	if err != nil {
		return err
	}

	// Delete the verrazzano-system namespace
	err = r.deleteNamespace(ctx, log, vzconst.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}
	return nil
}

// cleanupOld deltes the resources that used to be in the default namespace in earlier versions of Verrazzano.  This
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

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}

// Watch the jobs in the verrazzano-install for this vz resource.  The reconcile loop will be called
// when a job is updated.
func (r *Reconciler) watchJobs(namespace string, name string, log vzlog.VerrazzanoLogger) error {

	// Define a mapping to the Verrazzano resource
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}},
			}
		})

	// Watch job updates
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
	}

	// Watch jobs and trigger reconciles for Verrazzano resources when a job changes
	err := r.Controller.Watch(
		&source.Kind{Type: &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Namespace: getInstallNamespace()},
		}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		// Comment it if default predicate fun is used.
		p)
	if err != nil {
		return err
	}
	log.Debugf("Watching for jobs to activate reconcile for Verrazzano CR %s/%s", namespace, name)

	return nil
}

// Watch the pods in the keycloak namespace for this vz resource.  The loop to reconcile will be called
// when a pod is created.
func (r *Reconciler) watchPods(namespace string, name string, log vzlog.VerrazzanoLogger) error {
	// Define a mapping to the Verrazzano resource
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				}},
			}
		})

	// Watch pod create
	predicateFunc := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
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
			return true
		},
	}

	// Watch pods and trigger reconciles for Verrazzano resources when a pod is created
	err := r.Controller.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		predicateFunc)
	if err != nil {
		return err
	}
	log.Debugf("Watching for pods to activate reconcile for Verrazzano CR %s/%s", namespace, name)
	return nil
}

// initForVzResource will do initialization for the given Verrazzano resource.
// Clean up old resources from a 1.0 release where jobs, etc were in the default namespace
// Add a watch for each Verrazzano resource
func (r *Reconciler) initForVzResource(vz *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) (ctrl.Result, error) {
	if unitTesting {
		return ctrl.Result{}, nil
	}

	// Check if init done for this resource
	_, ok := initializedSet[vz.Name]
	if ok {
		return ctrl.Result{}, nil
	}

	// Add our finalizer if not already added
	if !vzstring.SliceContainsString(vz.ObjectMeta.Finalizers, finalizerName) {
		log.Debugf("Adding finalizer %s", finalizerName)
		vz.ObjectMeta.Finalizers = append(vz.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(context.TODO(), vz); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	// Cleanup old resources that might be left around when the install used to be done
	// in the default namespace
	if err := r.cleanupOld(context.TODO(), log, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	// Watch the jobs in the operator namespace for this VZ CR
	if err := r.watchJobs(vz.Namespace, vz.Name, log); err != nil {
		log.Errorf("Failed to set Job watch for Verrrazzano CR %s: %v", vz.Name, err)
		return newRequeueWithDelay(), err
	}

	// Watch pods in the keycloak namespace to handle recycle of the MySQL pod
	if err := r.watchPods(vz.Namespace, vz.Name, log); err != nil {
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

func (r *Reconciler) updateVerrazzano(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	err := r.Update(context.TODO(), vz)
	if err == nil {
		return nil
	}
	if ctrlerrors.IsUpdateConflict(err) {
		log.Info("Requeuing to get a new copy of the Verrazzano resource since the current one is outdated.")
	} else {
		log.Errorf("Failed to update Verrazzano resource :v", err)
	}
	// Return error so that reconcile gets called again
	return err
}

func (r *Reconciler) updateVerrazzanoStatus(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	err := r.Status().Update(context.TODO(), vz)
	if err == nil {
		return nil
	}
	if ctrlerrors.IsUpdateConflict(err) {
		log.Debugf("Requeuing to get a fresh copy of the Verrazzano resource since the current one is outdated.")
	} else {
		log.Errorf("Failed to update Verrazzano resource :v", err)
	}
	// Return error so that reconcile gets called again
	return err
}
