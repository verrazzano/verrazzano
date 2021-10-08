// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"k8s.io/apimachinery/pkg/util/rand"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzinstance"

	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/installjob"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
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

// Key into ConfigMap data for stored install Spec, the data for which will be used for update/upgrade purposes
const configDataKey = "spec"

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

	log.Info("Reconciler called")

	vz := &installv1alpha1.Verrazzano{}
	if err := r.Get(ctx, req.NamespacedName, vz); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Error getting the verrazzano resource - don't requeue.
		log.Errorf("Failed to fetch verrazzano resource: %v", err)
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

	// Process CR based on state
	switch vz.Status.State {
	case installv1alpha1.Failed:
		return r.FailedState(vz, log)
	case installv1alpha1.Installing:
		return r.InstallingState(vz, log)
	case installv1alpha1.Ready:
		return r.ReadyState(vz, log)
	case installv1alpha1.Uninstalling:
		return r.UninstallingState(vz, log)
	case installv1alpha1.Upgrading:
		return r.UpgradingState(vz, log)
	default:
		panic("Invalid Verrazzano contoller state")
	}
}

// ReadyState processes the CR while in the ready state
func (r *Reconciler) ReadyState(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	ctx := context.TODO()

	// Check if verrazzano resource is being deleted
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	// Pre-populate the component status fields
	result, err := r.initializeComponentStatus(vz)
	if err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}

	// If Verrazzano is installed see if upgrade is needed
	if isInstalled(vz.Status) {
		// If the version is specified and different than the current version of the installation
		// then proceed with upgrade
		if len(vz.Spec.Version) > 0 && vz.Spec.Version != vz.Status.Version {
			return r.reconcileUpgrade(log, vz)
		}
		if result, err := r.reconcileComponents(ctx, log, vz); err != nil {
			return newRequeueWithDelay(), err
		} else if shouldRequeue(result) {
			return result, nil
		}
		return ctrl.Result{}, nil
	}

	////************   TEMP CODE for TESTING ISTIO COMPONENT
	//
	//// Pre-create the Verrazzano System namespace if it doesn't already exist, before kicking off the install job,
	//// since it is needed for the subsequent step to syncLocalRegistration secret.
	//if err := r.createVerrazzanoSystemNamespace(ctx, log); err != nil {
	//	log.Errorf("Failed to create namespace %v: %v", vzconst.VerrazzanoSystemNamespace, err)
	//	return newRequeueWithDelay(), err
	//}
	//
	//if result, err := r.reconcileComponents(ctx, log, vz); err != nil {
	//	return newRequeueWithDelay(), err
	//} else if shouldRequeue(result) {
	//	return result, nil
	//}
	//
	//return ctrl.Result{}, nil
	//
	////*************   END TEMP CODE

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil {
		err := r.doesOCIDNSConfigSecretExist(vz)
		if err != nil {
			return newRequeueWithDelay(), err
		}
	}

	if err := r.createConfigMap(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	// Pre-create the Verrazzano System namespace if it doesn't already exist, before kicking off the install job,
	// since it is needed for the subsequent step to syncLocalRegistration secret.
	if err := r.createVerrazzanoSystemNamespace(ctx, log); err != nil {
		log.Errorf("Failed to create namespace %v: %v", vzconst.VerrazzanoSystemNamespace, err)
		return newRequeueWithDelay(), err
	}

	if err := r.createInstallJob(ctx, log, vz, buildConfigMapName(vz.Name)); err != nil {
		return newRequeueWithDelay(), err
	}

	// Sync the local cluster registration secret that allows the use of MCxyz resources on the
	// admin cluster without needing a VMC.
	if err := r.syncLocalRegistrationSecret(); err != nil {
		log.Errorf("Failed to sync the local registration secret: %v", err)
		return newRequeueWithDelay(), err
	}

	if result, err := r.reconcileComponents(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}

	// Create/update a configmap from spec for future comparison on update/upgrade
	if err := r.saveVerrazzanoSpec(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	}

	return ctrl.Result{}, nil
}

// InstallingState processes the CR while in the installing state
func (r *Reconciler) InstallingState(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	ctx := context.TODO()

	// Check if verrazzano resource is being deleted
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	if err := r.checkInstallJob(ctx, log, vz, buildConfigMapName(vz.Name)); err != nil {
		return newRequeueWithDelay(), err
	}

	if result, err := r.reconcileComponents(ctx, log, vz); err != nil {
		return newRequeueWithDelay(), err
	} else if shouldRequeue(result) {
		return result, nil
	}

	return ctrl.Result{}, nil
}

// UninstallingState processes the CR while in the uninstalling state
func (r *Reconciler) UninstallingState(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	ctx := context.TODO()

	// Update uninstall status
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.procDelete(ctx, log, vz)
	}

	return ctrl.Result{}, nil
}

// UpgradingState processes the CR while in the upgrading state
func (r *Reconciler) UpgradingState(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	return r.reconcileUpgrade(log, vz)
}

// FailedState only allows uninstall
func (r *Reconciler) FailedState(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (ctrl.Result, error) {
	ctx := context.TODO()

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
	serviceAccount := installjob.NewServiceAccount(getInstallNamespace(), buildServiceAccountName(vz.Name), imagePullSecrets, vz.Labels)

	// Check if the service account for running the scripts exist
	serviceAccountFound := &corev1.ServiceAccount{}
	log.Infof("Checking if install service account %s exist", buildServiceAccountName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildServiceAccountName(vz.Name), Namespace: getInstallNamespace()}, serviceAccountFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating install service account %s", buildServiceAccountName(vz.Name))
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
	binding := installjob.NewClusterRoleBinding(vz, buildClusterRoleBindingName(vz.Namespace, vz.Name), getInstallNamespace(), buildServiceAccountName(vz.Name))

	// Check if the cluster role binding for running the install scripts exist
	bindingFound := &rbacv1.ClusterRoleBinding{}
	log.Infof("Checking if install cluster role binding %s exist", binding.Name)
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

// createConfigMap creates a required config map for installation
func (r *Reconciler) createConfigMap(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Create the configmap resource that will contain installation configuration options
	configMap := installjob.NewConfigMap(getInstallNamespace(), buildConfigMapName(vz.Name), vz.Labels)

	// Check if the ConfigMap exists for running the install
	configMapFound := &corev1.ConfigMap{}
	log.Infof("Checking if install ConfigMap %s exist", configMap.Name)

	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, configMapFound)
	if err != nil && errors.IsNotFound(err) {
		vz = configFluentdExtraVolumeMounts(vz)
		config, err := installjob.GetInstallConfig(vz)
		if err != nil {
			return err
		}
		jsonEncoding, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{"config.json": string(jsonEncoding)}

		log.Infof("Creating install ConfigMap %s", configMap.Name)
		err = r.Create(ctx, configMap)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// deleteConfigMap deletes the config map used for installation
func (r *Reconciler) deleteConfigMap(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, namespace string) error {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      buildConfigMapName(vz.Name),
		},
	}
	err := r.Delete(ctx, &cm, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting ConfigMap %s: %v", cm.Name, err)
		return err
	}
	return nil
}

// deleteInstallJob Deletes the install job, which will also result in the install pod being deleted.
func (r *Reconciler) deleteInstallJob(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Check if the job for running the install scripts exist
	jobName := buildInstallJobName(vz.Name)
	jobFound := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: getInstallNamespace(),
			Name:      buildInstallJobName(vz.Name),
		},
	}
	log.Debugf("Checking if install job %s exist", jobName)
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: getInstallNamespace()}, jobFound)
	if err != nil {
		if !errors.IsNotFound(err) {
			// Got an error other than not found
			return err
		}
		// Job not found
		return nil
	}
	// Delete the Job in the foreground to ensure it's gone before continuing
	propagationPolicy := metav1.DeletePropagationForeground
	deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
	log.Infof("Install job %s in progress, deleting", jobName)
	return r.Delete(context.TODO(), jobFound, deleteOptions)
}

// createInstallJob creates the installation job
func (r *Reconciler) createInstallJob(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, configMapName string) error {
	// Define a new install job resource
	job := installjob.NewJob(
		&installjob.JobConfig{
			JobConfigCommon: k8s.JobConfigCommon{
				JobName:            buildInstallJobName(vz.Name),
				Namespace:          getInstallNamespace(),
				Labels:             vz.Labels,
				ServiceAccountName: buildServiceAccountName(vz.Name),
				JobImage:           os.Getenv("VZ_INSTALL_IMAGE"),
				DryRun:             r.DryRun,
			},
			ConfigMapName: configMapName,
		})

	// Check if the job for running the install scripts exist
	jobFound := &batchv1.Job{}
	log.Infof("Checking if install job %s exist", buildInstallJobName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildInstallJobName(vz.Name), Namespace: getInstallNamespace()}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating install job %s, dry-run=%v", buildInstallJobName(vz.Name), r.DryRun)
		err = r.Create(ctx, job)
		if err != nil {
			return err
		}

		// Add our finalizer if not already added
		if !containsString(vz.ObjectMeta.Finalizers, finalizerName) {
			log.Infof("Adding finalizer %s", finalizerName)
			vz.ObjectMeta.Finalizers = append(vz.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, vz); err != nil {
				return err
			}
		}

		// Delete leftover uninstall job if we find one.
		err = r.cleanupUninstallJob(buildUninstallJobName(vz.Name), getInstallNamespace(), log)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := installv1alpha1.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	vz.Status.Version = bomSemVer.ToString()
	err = r.setInstallCondition(log, jobFound, vz)

	return err
}

// checkInstallJob checks the installation job
func (r *Reconciler) checkInstallJob(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, configMapName string) error {
	// Check if the job for running the install scripts exist
	jobFound := &batchv1.Job{}
	log.Infof("Checking if install job %s exist", buildInstallJobName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildInstallJobName(vz.Name), Namespace: getInstallNamespace()}, jobFound)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Update condition and status
	err = r.setInstallCondition(log, jobFound, vz)
	return err
}

// cleanupUninstallJob checks for the existence of a stale uninstall job and deletes the job if one is found
func (r *Reconciler) cleanupUninstallJob(jobName string, namespace string, log *zap.SugaredLogger) error {
	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Infof("Checking if stale uninstall job %s exists", jobName)
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: namespace}, jobFound)
	if err == nil {
		log.Infof("Deleting stale uninstall job %s", jobName)
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
		For(&installv1alpha1.Verrazzano{}).
		// The GenerateChangedPredicate will skip update events that have no change in the object's metadata.generation
		// field.  Any updates to the status or metadata do not cause the metadata.generation to be changed and
		// therefore the reconciler will not be called.
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Build(r)
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
	log.Infof("Checking if uninstall job %s exist", buildUninstallJobName(vz.Name))
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

	err = r.setUninstallCondition(log, jobFound, vz)
	if err != nil {
		return err
	}

	return nil
}

// buildInstallJobName returns the name of an install job based on verrazzano resource name.
func buildInstallJobName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// buildUninstallJobName returns the name of an uninstall job based on verrazzano resource name.
func buildUninstallJobName(name string) string {
	return fmt.Sprintf("verrazzano-uninstall-%s", name)
}

// buildServiceAccountName returns the service account name for jobs based on verrazzano resource name.
func buildServiceAccountName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// buildClusterRoleBindingName returns the ClusgterRoleBinding name for jobs based on verrazzano resource name.
func buildClusterRoleBindingName(namespace string, name string) string {
	return fmt.Sprintf("verrazzano-install-%s-%s", namespace, name)
}

// buildConfigMapName returns the name of a config map for an install job based on verrazzano resource name.
func buildConfigMapName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// buildInternalConfigMapName returns the name of the internal configmap associated with an install resource.
func buildInternalConfigMapName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s-internal", name)
}

// updateStatus updates the status in the verrazzano CR
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
	log.Infof("Setting verrazzano resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update verrazzano resource status: %v", err)
		return err
	}
	return nil
}

// updateState updates the status state in the verrazzano CR
func (r *Reconciler) updateState(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano, state installv1alpha1.StateType) error {
	// Set the state of resource
	cr.Status.State = state
	log.Infof("Setting verrazzano state: %v", cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update verrazzano resource status: %v", err)
		return err
	}
	return nil
}

func (r *Reconciler) updateComponentStatus(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano, componentName string, message string, conditionType installv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}

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

	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus, condition)

	// Set the state of resource
	componentStatus.State = checkCondtitionType(conditionType)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update verrazzano resource status: %v", err)
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
	log.Infof("Adding %s resource newCondition: %v", compStatus.Name, newCondition.Type)
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

// setInstallCondition sets the verrazzano resource condition in status for install
func (r *Reconciler) setInstallCondition(log *zap.SugaredLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
	// If the job has succeeded or failed add the appropriate condition
	if job.Status.Succeeded != 0 || job.Status.Failed != 0 {
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.InstallComplete || condition.Type == installv1alpha1.InstallFailed {
				return nil
			}
		}
		var message string
		var conditionType installv1alpha1.ConditionType
		if job.Status.Succeeded == 1 {
			message = "Verrazzano install completed successfully"
			if checkComponentReadyState(vz) {
				// Set install complete IFF all subcomponent status' are "Ready"
				conditionType = installv1alpha1.InstallComplete
				log.Info(message)
				vz.Status.VerrazzanoInstance = vzinstance.GetInstanceInfo(r.Client, vz)
			}
		} else {
			message = "Verrazzano install failed to complete"
			conditionType = installv1alpha1.InstallFailed
			log.Error(message)
		}
		return r.updateStatus(log, vz, message, conditionType)
	}

	// Add the install started condition if not already added
	for _, condition := range vz.Status.Conditions {
		if condition.Type == installv1alpha1.InstallStarted {
			return nil
		}
	}

	return r.updateStatus(log, vz, "Verrazzano install in progress", installv1alpha1.InstallStarted)
}

// checkComponentReadyState returns true if all component-level status' are "Ready"
func checkComponentReadyState(vz *installv1alpha1.Verrazzano) bool {
	for _, compStatus := range vz.Status.Components {
		if compStatus.State != installv1alpha1.Disabled && compStatus.State != installv1alpha1.Ready {
			return false
		}
	}
	return true
}

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based in stall.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	if cr.Status.Components != nil {
		return ctrl.Result{}, nil
	}
	cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			cr.Status.Components[comp.Name()] = &installv1alpha1.ComponentStatusDetails{
				Name:  comp.Name(),
				State: installv1alpha1.Disabled,
			}
		}
	}
	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	return ctrl.Result{Requeue: true}, err
}

// setUninstallCondition sets the verrazzano resource condition in status for uninstall
func (r *Reconciler) setUninstallCondition(log *zap.SugaredLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
	// If the job has succeeded or failed add the appropriate condition
	if job.Status.Succeeded != 0 || job.Status.Failed != 0 {
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.UninstallComplete || condition.Type == installv1alpha1.UninstallFailed {
				return nil
			}
		}

		// Remove the owner reference so that the install job is not deleted when the verrazzano resource is deleted
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

// saveInstallSpec Saves the install spec in a configmap to use with upgrade/updates later on
func (r *Reconciler) saveVerrazzanoSpec(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (err error) {
	installSpecBytes, err := yaml.Marshal(vz.Spec)
	if err != nil {
		return err
	}
	installSpec := base64.StdEncoding.EncodeToString(installSpecBytes)
	installConfig, err := r.getInternalConfigMap(ctx, vz)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		configMapName := buildInternalConfigMapName(vz.Name)
		configData := make(map[string]string)
		configData[configDataKey] = installSpec
		// Create the configmap and set the owner reference to the VZ installer resource for garbage collection
		installConfig = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: getInstallNamespace(),
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: vz.APIVersion,
					Kind:       vz.Kind,
					Name:       vz.Name,
					UID:        vz.UID,
				}},
			},
			Data: configData,
		}
		err := r.Create(ctx, installConfig)
		if err != nil {
			log.Errorf("Unable to create installer config map %s: %v", configMapName, err)
			return err
		}
	} else {
		// Update the configmap if the data has changed
		currentConfigData := installConfig.Data[configDataKey]
		if currentConfigData != installSpec {
			installConfig.Data[configDataKey] = installSpec
			return r.Update(ctx, installConfig)
		}
	}
	return nil
}

// getSavedInstallSpec Returns the saved Verrazzano resource Spec field from the internal ConfigMap, or an error if it can't be restored
func (r *Reconciler) getSavedInstallSpec(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (*installv1alpha1.VerrazzanoSpec, error) {
	configMap, err := r.getInternalConfigMap(ctx, vz)
	if err != nil {
		log.Warnf("No saved configuration found for install spec for %s", vz.Name)
		return nil, err
	}
	storedSpec := &installv1alpha1.VerrazzanoSpec{}
	if specData, ok := configMap.Data[configDataKey]; ok {
		decodeBytes, err := base64.StdEncoding.DecodeString(specData)
		if err != nil {
			log.Errorf("Error decoding saved install spec for %s", vz.Name)
			return nil, err
		}
		if err := yaml.Unmarshal(decodeBytes, storedSpec); err != nil {
			log.Errorf("Error unmarshalling saved install spec for %s", vz.Name)
			return nil, err
		}
	}
	return storedSpec, nil
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

// createVerrazzanoSystemNamespace creates the verrazzano system namespace if it does not already exist
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
		log.Infof("Namespace %v was successfully created", vzconst.VerrazzanoSystemNamespace)
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
		subdomain = "default"
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

func configFluentdExtraVolumeMounts(vz *installv1alpha1.Verrazzano) *installv1alpha1.Verrazzano {
	varLog := "/var/log/containers/"
	var files []string
	filepath.Walk(varLog, func(path string, info os.FileInfo, err error) error {
		if info != nil {
			files = append(files, readLink(path, info)...)
		}
		return nil
	})
	return addFluentdExtraVolumeMounts(files, vz)
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

// Process the Verrazzano resource deletion
func (r *Reconciler) procDelete(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Finalizer is present, so lets do the uninstall
	if containsString(vz.ObjectMeta.Finalizers, finalizerName) {
		// Delete the install job if it exists, cancelling any running install jobs before uninstalling
		if err := r.deleteInstallJob(log, vz); err != nil {
			log.Errorf("Failed creating the install job: %v", err)
			return newRequeueWithDelay(), err
		}
		// Create the uninstall job if it doesn't exist
		if err := r.createUninstallJob(log, vz); err != nil {
			log.Errorf("Failed creating the uninstall job: %v", err)
			return newRequeueWithDelay(), err
		}

		// Remove the finalizer and update the verrazzano resource if the uninstall has finished.
		for _, condition := range vz.Status.Conditions {
			if condition.Type == installv1alpha1.UninstallComplete || condition.Type == installv1alpha1.UninstallFailed {
				err := r.cleanup(ctx, log, vz)
				if err != nil {
					return newRequeueWithDelay(), err
				}

				// All install related resources have been deleted, delete the finalizer so that the Verrazzano
				// resource can get removed from etcd.
				log.Infof("Removing finalizer %s", finalizerName)
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

	// Delete the install config map
	err = r.deleteConfigMap(ctx, log, vz, getInstallNamespace())
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

	// Delete the install config map
	err = r.deleteConfigMap(ctx, log, vz, vzconst.DefaultNamespace)
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

	// Define a mapping to the verrazzano resource
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
	log.Infof("Watching for jobs to activate reconcile for Verrrazzano CR %s/%s", namespace, name)

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
