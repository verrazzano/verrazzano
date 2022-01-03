// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"

	cmapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/uninstalljob"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
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

// Reconcile will reconcile the CR
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;watch;list;create;update;delete
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))

	// Add cert-manager components to the scheme
	cmapiv1.AddToScheme(r.Scheme)

	log.Debugf("Reconciler called")

	vz := &installv1alpha1.Verrazzano{}
	if err := r.Get(ctx, req.NamespacedName, vz); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the Verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Error getting the Verrazzano resource - don't requeue.
		log.Errorf("Failed to fetch Verrazzano resource: %v", err)
		return reconcile.Result{}, err
	}

	// Initialize once for this Verrazzano resource when the operator starts
	result, err := r.initForVzResource(vz, log)
	if err != nil {
		log.Errorf("unable to set watch for Job resource: %v", err)
		return result, err
	}
	if shouldRequeue(result) {
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
		r.updateState(log, vz, installv1alpha1.Ready)
		return reconcile.Result{Requeue: true}, nil
	}

	spiContext, err := spi.NewContext(log, r, vz, r.DryRun)
	if err != nil {
		log.Errorf("Could not create component context: %v", err)
		return newRequeueWithDelay(), err
	}

	// Process CR based on state
	switch vz.Status.State {
	case installv1alpha1.Failed:
		return r.FailedState(spiContext)
	case installv1alpha1.Installing:
		return r.InstallingState(spiContext)
	case installv1alpha1.Ready:
		return r.ReadyState(spiContext)
	case installv1alpha1.Uninstalling:
		return r.UninstallingState(spiContext)
	case installv1alpha1.Upgrading:
		return r.UpgradingState(spiContext)
	default:
		panic("Invalid Verrazzano contoller state")
	}
}

// ReadyState processes the CR while in the ready state
func (r *Reconciler) ReadyState(spiCtx spi.ComponentContext) (ctrl.Result, error) {
	log := spiCtx.Log()
	actualCR := spiCtx.ActualCR()

	log.Debugf("enter ReadyState")
	ctx := context.TODO()

	// Check if Verrazzano resource is being deleted
	if !actualCR.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, actualCR)
	}

	// Pre-populate the component status fields
	result, err := r.initializeComponentStatus(log, actualCR)
	if err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}

	// If Verrazzano is installed see if upgrade is needed
	if isInstalled(actualCR.Status) {
		// If the version is specified and different than the current version of the installation
		// then proceed with upgrade
		if len(actualCR.Spec.Version) > 0 && actualCR.Spec.Version != actualCR.Status.Version {
			return r.reconcileUpgrade(log, actualCR)
		}
		if result, err := r.reconcileComponents(ctx, spiCtx); err != nil {
			return newRequeueWithDelay(), err
		} else if shouldRequeue(result) {
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
		log.Errorf("Failed to create namespace %v: %v", vzconst.VerrazzanoSystemNamespace, err)
		return newRequeueWithDelay(), err
	}

	// Sync the local cluster registration secret that allows the use of MCxyz resources on the
	// admin cluster without needing a VMC.
	if err := r.syncLocalRegistrationSecret(); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return newRequeueWithDelay(), err
	}

	// Change the state back to ready if install complete otherwise requeue
	done, err := r.checkInstallComplete(spiCtx)
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

// InstallingState processes the CR while in the installing state
func (r *Reconciler) InstallingState(spiCtx spi.ComponentContext) (ctrl.Result, error) {
	log := spiCtx.Log()
	actualCR := spiCtx.ActualCR()
	log.Debugf("enter InstallingState")
	ctx := context.TODO()

	// Check if Verrazzano resource is being deleted
	if !actualCR.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, actualCR)
	}

	if result, err := r.reconcileComponents(ctx, spiCtx); err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}

	// Change the state back to ready if install complete otherwise requeue
	done, err := r.checkInstallComplete(spiCtx)
	if !done || err != nil {
		return newRequeueWithDelay(), err
	}
	return ctrl.Result{}, nil
}

// UninstallingState processes the CR while in the uninstalling state
func (r *Reconciler) UninstallingState(spiCtx spi.ComponentContext) (ctrl.Result, error) {
	vz := spiCtx.ActualCR()
	log := spiCtx.Log()
	log.Debugf("enter UninstallingState")
	ctx := context.TODO()

	// Update uninstall status
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	return ctrl.Result{}, nil
}

// UpgradingState processes the CR while in the upgrading state
func (r *Reconciler) UpgradingState(spiCtx spi.ComponentContext) (ctrl.Result, error) {
	vz := spiCtx.ActualCR()
	log := spiCtx.Log()
	log.Debugf("enter UpgradingState")

	if result, err := r.reconcileUpgrade(log, vz); err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}
	// Upgrade should always requeue to ensure that reconciler runs post upgrade to install
	// components that may have been waiting for upgrade
	return newRequeueWithDelay(), nil
}

// FailedState only allows uninstall
func (r *Reconciler) FailedState(spiCtx spi.ComponentContext) (ctrl.Result, error) {
	vz := spiCtx.ActualCR()
	log := spiCtx.Log()
	log.Debugf("enter FailedState")
	ctx := context.TODO()

	// Determine if the user specified to retry upgrade
	retry, err := r.retryUpgrade(ctx, vz)
	if err != nil {
		log.Errorf("Failed to update the annotations: %v", err)
		return newRequeueWithDelay(), err
	}

	if retry {
		// Log the retry and set the StateType to ready, then requeue
		log.Debugf("Restart Version annotation has changed, retrying upgrade")
		err = r.updateState(log, vz, installv1alpha1.Ready)
		if err != nil {
			log.Errorf("Failed to update the state to ready: %v", err)
			return newRequeueWithDelay(), err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	// Update uninstall status
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
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
func (r *Reconciler) createServiceAccount(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
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
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// deleteServiceAccount deletes the service account used for install
func (r *Reconciler) deleteServiceAccount(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, namespace string) error {
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
func (r *Reconciler) createClusterRoleBinding(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new cluster role binding resource
	binding := rbac.NewClusterRoleBinding(vz, buildClusterRoleBindingName(vz.Namespace, vz.Name), getInstallNamespace(), buildServiceAccountName(vz.Name))

	// Check if the cluster role binding for running the install scripts exist
	bindingFound := &rbacv1.ClusterRoleBinding{}
	log.Debugf("Checking if install cluster role binding %s exist", binding.Name)
	err := r.Get(ctx, types.NamespacedName{Name: binding.Name, Namespace: binding.Namespace}, bindingFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating install cluster role binding %s", binding.Name)
		err = r.Create(ctx, binding)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// deleteClusterRoleBinding deletes the cluster role binding
func (r *Reconciler) deleteClusterRoleBinding(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
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
func (r *Reconciler) checkInstallComplete(spiCtx spi.ComponentContext) (bool, error) {
	log := spiCtx.Log()
	actualCR := spiCtx.ActualCR()
	ready, err := r.checkComponentReadyState(spiCtx)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	// Set install complete IFF all subcomponent status' are "Ready"
	message := "Verrazzano install completed successfully"
	// Status update must be performed on the actual CR read from K8S
	return true, r.updateStatus(log, actualCR, message, installv1alpha1.InstallComplete)
}

// cleanupUninstallJob checks for the existence of a stale uninstall job and deletes the job if one is found
func (r *Reconciler) cleanupUninstallJob(jobName string, namespace string, log *zap.SugaredLogger) error {
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
func (r *Reconciler) deleteNamespace(ctx context.Context, log *zap.SugaredLogger, namespace string) error {
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

func (r *Reconciler) createUninstallJob(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
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
		log.Debugf("Creating uninstall job %s, dry-run=%v", buildUninstallJobName(vz.Name), r.DryRun)
		err = r.Create(context.TODO(), job)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

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

// updateStatus updates the status in the Verrazzano CR
func (r *Reconciler) updateStatus(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType) error {
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
	cr.Status.State = checkCondtitionType(conditionType)
	log.Infof("Setting Verrazzano resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update Verrazzano resource status: %v", err)
		return err
	}
	return nil
}

// updateState updates the status state in the Verrazzano CR
func (r *Reconciler) updateState(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano, state installv1alpha1.StateType) error {
	// Set the state of resource
	cr.Status.State = state
	log.Infof("Setting Verrazzano state: %v", cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update Verrazzano resource status: %v", err)
		return err
	}
	return nil
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
	if conditionType == installv1alpha1.InstallComplete {
		cr.Status.VerrazzanoInstance = vzinstance.GetInstanceInfo(compContext)
	}
	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus, condition)

	// Set the state of resource
	componentStatus.State = checkCondtitionType(conditionType)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update Verrazzano resource status: %v", err)
		return err
	}
	return nil
}

func appendConditionIfNecessary(log *zap.SugaredLogger, compStatus *installv1alpha1.ComponentStatusDetails, newCondition installv1alpha1.Condition) []installv1alpha1.Condition {
	for _, existingCondition := range compStatus.Conditions {
		if existingCondition.Type == newCondition.Type {
			return compStatus.Conditions
		}
	}
	log.Debugf("Adding %s resource newCondition: %v", compStatus.Name, newCondition.Type)
	return append(compStatus.Conditions, newCondition)
}

func checkCondtitionType(currentCondition installv1alpha1.ConditionType) installv1alpha1.StateType {
	switch currentCondition {
	case installv1alpha1.PreInstall:
		return installv1alpha1.PreInstalling
	case installv1alpha1.InstallStarted:
		return installv1alpha1.Installing
	case installv1alpha1.UninstallStarted:
		return installv1alpha1.Uninstalling
	case installv1alpha1.UpgradeStarted:
		return installv1alpha1.Upgrading
	case installv1alpha1.UninstallComplete:
		return installv1alpha1.Ready
	case installv1alpha1.InstallFailed, installv1alpha1.UpgradeFailed, installv1alpha1.UninstallFailed:
		return installv1alpha1.Failed
	}
	// Return ready for installv1alpha1.InstallComplete, installv1alpha1.UpgradeComplete
	return installv1alpha1.Ready
}

// setInstallStartedCondition
func (r *Reconciler) setInstallingState(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := installv1alpha1.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	vz.Status.Version = bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano install in progress", installv1alpha1.InstallStarted)
}

// checkComponentReadyState returns true if all component-level status' are "Ready" for enabled components
func (r *Reconciler) checkComponentReadyState(context spi.ComponentContext) (bool, error) {
	cr := context.ActualCR()
	if unitTesting {
		for _, compStatus := range cr.Status.Components {
			if compStatus.State != installv1alpha1.Disabled && compStatus.State != installv1alpha1.Ready {
				return false, nil
			}
		}
		return true, nil
	}

	// Return false if any enabled component is not ready
	for _, comp := range registry.GetComponents() {
		if comp.IsEnabled(context) && cr.Status.Components[comp.Name()].State != installv1alpha1.Ready {
			return false, nil
		}
	}
	return true, nil
}

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based in stall.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	if cr.Status.Components != nil {
		return ctrl.Result{}, nil
	}

	log.Debugf("initializeComponentStatus for all components")
	cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)

	newContext, err := spi.NewContext(log, r, cr, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			// If the component is installed then mark it as ready
			compContext := newContext.For(comp.Name()).Operation(vzconst.InitializeOperation)
			state := installv1alpha1.Disabled
			if !unitTesting {
				installed, err := comp.IsInstalled(compContext)
				if err != nil {
					log.Errorf("IsInstalled error for component %s: %s", comp.Name(), err)
					return newRequeueWithDelay(), err
				}
				if installed {
					state = installv1alpha1.Ready
				}
			}
			cr.Status.Components[comp.Name()] = &installv1alpha1.ComponentStatusDetails{
				Name:  comp.Name(),
				State: state,
			}
		}
	}
	// Update the status
	err = r.Status().Update(context.TODO(), cr)
	return ctrl.Result{Requeue: true}, err
}

// setUninstallCondition sets the Verrazzano resource condition in status for uninstall
func (r *Reconciler) setUninstallCondition(log *zap.SugaredLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
	// If the job has succeeded or failed add the appropriate condition
	if job.Status.Succeeded != 0 || job.Status.Failed != 0 {
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.UninstallComplete || condition.Type == installv1alpha1.UninstallFailed {
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
			message = "Verrazzano uninstall completed successfully"
			conditionType = installv1alpha1.UninstallComplete
			log.Info(message)
		} else {
			message = "Verrazzano uninstall failed to complete"
			conditionType = installv1alpha1.UninstallFailed
			log.Error(message)
		}
		return r.updateStatus(log, vz, message, conditionType)
	}

	// Add the uninstall started condition if not already added
	for _, condition := range vz.Status.Conditions {
		if condition.Type == installv1alpha1.UninstallStarted {
			return nil
		}
	}

	return r.updateStatus(log, vz, "Verrazzano uninstall in progress", installv1alpha1.UninstallStarted)
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
func (r *Reconciler) createVerrazzanoSystemNamespace(ctx context.Context, log *zap.SugaredLogger) error {

	// First check if VZ system namespace exists. If not, create it.
	var vzSystemNS corev1.Namespace
	err := r.Get(ctx, types.NamespacedName{Name: vzconst.VerrazzanoSystemNamespace}, &vzSystemNS)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		vzSystemNS.Name = vzconst.VerrazzanoSystemNamespace
		vzSystemNS.Labels, _ = mergeMaps(nil, systemNamespaceLabels)
		if err := r.Create(ctx, &vzSystemNS); err != nil {
			return err
		}
		log.Debugf("Namespace %v was successfully created", vzconst.VerrazzanoSystemNamespace)
		return nil
	}
	// Namespace exists, see if we need to add the label
	var updated bool
	vzSystemNS.Labels, updated = mergeMaps(vzSystemNS.Labels, systemNamespaceLabels)
	if !updated {
		return nil
	}
	if err := r.Update(ctx, &vzSystemNS); err != nil {
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

// containsString checks for a string in a slice of strings
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removeString removes a string from a slice of strings
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// buildDomain Build the DNS Domain from the current install
func buildDomain(c client.Client, vz *installv1alpha1.Verrazzano) (string, error) {
	subdomain := vz.Spec.EnvironmentName
	if len(subdomain) == 0 {
		subdomain = vzconst.DefaultEnvironmentName
	}
	baseDomain, err := buildDomainSuffix(c, vz)
	if err != nil {
		return "", err
	}
	domain := subdomain + "." + baseDomain
	return domain, nil
}

// buildDomainSuffix Get the configured domain suffix, or compute the nip.io domain
func buildDomainSuffix(c client.Client, vz *installv1alpha1.Verrazzano) (string, error) {
	dns := vz.Spec.Components.DNS
	if dns != nil && dns.OCI != nil {
		return dns.OCI.DNSZoneName, nil
	}
	if dns != nil && dns.External != nil {
		return dns.External.Suffix, nil
	}
	ipAddress, err := getIngressIP(c)
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
func getIngressIP(c client.Client) (string, error) {
	const nginxIngressController = "ingress-controller-ingress-nginx-controller"
	const nginxNamespace = "ingress-nginx"

	nginxService := corev1.Service{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: nginxIngressController, Namespace: nginxNamespace}, &nginxService)
	if err != nil {
		return "", err
	}
	if nginxService.Spec.Type == corev1.ServiceTypeLoadBalancer {
		nginxIngress := nginxService.Status.LoadBalancer.Ingress
		if len(nginxIngress) == 0 {
			// In case of OLCNE, need to obtain the External IP from the Spec
			if len(nginxService.Spec.ExternalIPs) == 0 {
				return "", fmt.Errorf("%s is missing External IP address", nginxService.Name)
			}
			return nginxService.Spec.ExternalIPs[0], nil
		}
		return nginxIngress[0].IP, nil
	} else if nginxService.Spec.Type == corev1.ServiceTypeNodePort {
		return "127.0.0.1", nil
	}
	return "", fmt.Errorf("Unsupported service type %s for NGINX ingress", string(nginxService.Spec.Type))
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

func readLink(path string, info os.FileInfo) []string {
	var files []string
	if info.Mode()&os.ModeSymlink != 0 {
		dest, err := os.Readlink(path)
		if err == nil {
			files = append(files, dest)
			destInfo, err := os.Lstat(dest)
			if err == nil {
				files = append(files, readLink(dest, destInfo)...)
			}
		}
	}
	return files
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
func (r *Reconciler) procDelete(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Finalizer is present, so lets do the uninstall
	if containsString(vz.ObjectMeta.Finalizers, finalizerName) {
		// Create the uninstall job if it doesn't exist
		if err := r.createUninstallJob(log, vz); err != nil {
			log.Errorf("Failed creating the uninstall job: %v", err)
			return newRequeueWithDelay(), err
		}

		// Remove the finalizer and update the Verrazzano resource if the uninstall has finished.
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.UninstallComplete || condition.Type == installv1alpha1.UninstallFailed {
				err := r.cleanup(ctx, log, vz)
				if err != nil {
					return newRequeueWithDelay(), err
				}

				// All install related resources have been deleted, delete the finalizer so that the Verrazzano
				// resource can get removed from etcd.
				log.Debugf("Removing finalizer %s", finalizerName)
				vz.ObjectMeta.Finalizers = removeString(vz.ObjectMeta.Finalizers, finalizerName)
				err = r.Update(ctx, vz)
				if err != nil {
					return newRequeueWithDelay(), err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// Cleanup the resources left over from install and uninstall
func (r *Reconciler) cleanup(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
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
func (r *Reconciler) cleanupOld(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
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
	var seconds = rand.IntnRange(3, 5)
	delaySecs := time.Duration(seconds) * time.Second
	return ctrl.Result{Requeue: true, RequeueAfter: delaySecs}
}

// Return true if requeue is needed
func shouldRequeue(r ctrl.Result) bool {
	return r.Requeue || r.RequeueAfter > 0
}

// Watch the jobs in the verrazzano-install for this vz resource.  The reconcile loop will be called
// when a job is updated.
func (r *Reconciler) watchJobs(namespace string, name string, log *zap.SugaredLogger) error {

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

// initForVzResource will do initialization for the given Verrazzano resource.
// Clean up old resources from a 1.0 release where jobs, etc were in the default namespace
// Add a watch for each Verrazzano resource
func (r *Reconciler) initForVzResource(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	if unitTesting {
		return ctrl.Result{}, nil
	}

	// Check if init done for this resource
	_, ok := initializedSet[vz.Name]
	if ok {
		return ctrl.Result{}, nil
	}

	// Add our finalizer if not already added
	if !containsString(vz.ObjectMeta.Finalizers, finalizerName) {
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
		log.Errorf("unable to set Job watch for Verrrazzano CR %s: %v", vz.Name, err)
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
