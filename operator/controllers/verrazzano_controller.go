// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/operator/internal/k8s"
	"os"
	"time"

	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/internal/installjob"
	"github.com/verrazzano/verrazzano/operator/internal/uninstalljob"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

// VerrazzanoReconciler reconciles a Verrazzano object
type VerrazzanoReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

// Name of finalizer
const finalizerName = "install.verrazzano.io"

// Key into ConfigMap data for stored install Spec, the data for which will be used for update/upgrade purposes
const configDataKey = "spec"

// Reconcile will reconcile the CR
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;watch;list;create;update;delete
func (r *VerrazzanoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
		log.Error(err, "Failed to fetch verrazzano resource")
		return reconcile.Result{}, err
	}

	// The verrazzano resource is being deleted
	if !vz.ObjectMeta.DeletionTimestamp.IsZero() {
		// Finalizer is present, so lets do the uninstall
		if containsString(vz.ObjectMeta.Finalizers, finalizerName) {
			if err := r.createUninstallJob(log, vz); err != nil {
				// If fail to start the uninstall, return with error so that it can be retried
				return reconcile.Result{}, err
			}

			// Remove the finalizer and update the verrazzano resource if the uninstall has finished.
			for _, condition := range vz.Status.Conditions {
				if condition.Type == installv1alpha1.UninstallComplete || condition.Type == installv1alpha1.UninstallFailed {
					log.Infof("Removing finalizer %s", finalizerName)
					vz.ObjectMeta.Finalizers = removeString(vz.ObjectMeta.Finalizers, finalizerName)
					if err := r.Update(ctx, vz); err != nil {
						return reconcile.Result{}, err
					}
				}
			}
		}
		return reconcile.Result{}, nil
	}

	// If Verrazzano is installed see if upgrade is needed
	if isInstalled(vz.Status) {
		// If the version is specified and different than the current version of the installation
		// then proceed with upgrade
		if len(vz.Spec.Version) > 0 && vz.Spec.Version != vz.Status.Version {
			return r.reconcileUpgrade(log, req, vz)
		}
		// nothing to do, installation already at target version
		return ctrl.Result{}, nil
	}

	if err := r.createServiceAccount(ctx, log, vz); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.createClusterRoleBinding(ctx, log, vz); err != nil {
		return reconcile.Result{}, err
	}

	// if an OCI DNS installation, make sure the secret required exists before proceeding
	if vz.Spec.Components.DNS.OCI != (installv1alpha1.OCI{}) {
		err := r.doesOCIDNSConfigSecretExist(vz)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	err := r.createConfigMap(ctx, log, vz)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.createInstallJob(ctx, log, vz, buildConfigMapName(vz.Name)); err != nil {
		return reconcile.Result{}, err
	}

	// Create/update a configmap from spec for future comparison on update/upgrade
	if err := r.saveVerrazzanoSpec(ctx, log, vz); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, err
}

func (r *VerrazzanoReconciler) doesOCIDNSConfigSecretExist(vz *installv1alpha1.Verrazzano) error {
	// ensure the secret exists before proceeding
	secret := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: vz.Spec.Components.DNS.OCI.OCIConfigSecret, Namespace: "default"}, secret)
	if err != nil {
		return err
	}
	return nil
}

// createServiceAccount creates a required service account
func (r *VerrazzanoReconciler) createServiceAccount(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new service account resource
	serviceAccount := installjob.NewServiceAccount(vz.Namespace, buildServiceAccountName(vz.Name), os.Getenv("IMAGE_PULL_SECRET"), vz.Labels)

	// Set verrazzano resource as the owner and controller of the service account resource.
	// This reference will result in the service account resource being deleted when the verrazzano CR is deleted.
	if err := controllerutil.SetControllerReference(vz, serviceAccount, r.Scheme); err != nil {
		return err
	}

	// Check if the service account for running the scripts exist
	serviceAccountFound := &corev1.ServiceAccount{}
	log.Infof("Checking if install service account %s exist", buildServiceAccountName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildServiceAccountName(vz.Name), Namespace: vz.Namespace}, serviceAccountFound)
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

// createClusterRoleBinding creates a required cluster role binding
func (r *VerrazzanoReconciler) createClusterRoleBinding(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new cluster role binding resource
	clusterRoleBinding := installjob.NewClusterRoleBinding(vz, buildClusterRoleBindingName(vz.Namespace, vz.Name), buildServiceAccountName(vz.Name))

	// Check if the cluster role binding for running the install scripts exist
	clusterRoleBindingFound := &rbacv1.ClusterRoleBinding{}
	log.Infof("Checking if install cluster role binding %s exist", clusterRoleBinding.Name)
	err := r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: clusterRoleBinding.Namespace}, clusterRoleBindingFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating install cluster role binding %s", clusterRoleBinding.Name)
		err = r.Create(ctx, clusterRoleBinding)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// createConfigMap creates a required config map for installation
func (r *VerrazzanoReconciler) createConfigMap(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Create the configmap resource that will contain installation configuration options
	configMap := installjob.NewConfigMap(vz.Namespace, buildConfigMapName(vz.Name), vz.Labels)

	// Set the verrazzano resource as the owner and controller of the configmap
	err := controllerutil.SetControllerReference(vz, configMap, r.Scheme)
	if err != nil {
		return err
	}

	// Check if the ConfigMap exists for running the install
	configMapFound := &corev1.ConfigMap{}
	log.Infof("Checking if install ConfigMap %s exist", configMap.Name)

	err = r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, configMapFound)
	if err != nil && errors.IsNotFound(err) {
		config := installjob.GetInstallConfig(vz)
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

// createInstallJob creates the installation job
func (r *VerrazzanoReconciler) createInstallJob(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, configMapName string) error {
	// Define a new install job resource
	job := installjob.NewJob(
		&installjob.JobConfig{
			JobConfigCommon: k8s.JobConfigCommon{
				JobName:            buildInstallJobName(vz.Name),
				Namespace:          vz.Namespace,
				Labels:             vz.Labels,
				ServiceAccountName: buildServiceAccountName(vz.Name),
				JobImage:           os.Getenv("VZ_INSTALL_IMAGE"),
				DryRun:             r.DryRun,
			},
			ConfigMapName: configMapName,
		})

	// Set verrazzano resource as the owner and controller of the job resource.
	// This reference will result in the job resource being deleted when the verrazzano CR is deleted.
	if err := controllerutil.SetControllerReference(vz, job, r.Scheme); err != nil {
		return err
	}
	// Check if the job for running the install scripts exist
	jobFound := &batchv1.Job{}
	log.Infof("Checking if install job %s exist", buildInstallJobName(vz.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildInstallJobName(vz.Name), Namespace: vz.Namespace}, jobFound)
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
		err = r.cleanupUninstallJob(buildUninstallJobName(vz.Name), vz.Namespace, log)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	err = r.setInstallCondition(log, jobFound, vz)

	return err
}

// cleanupUninstallJob checks for the existence of a stale uninstall job and deletes the job if one is found
func (r *VerrazzanoReconciler) cleanupUninstallJob(jobName string, namespace string, log *zap.SugaredLogger) error {
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

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Verrazzano{}).
		Build(r)
	return err
}

func (r *VerrazzanoReconciler) createUninstallJob(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) error {
	// Define a new uninstall job resource
	job := uninstalljob.NewJob(
		&uninstalljob.JobConfig{
			JobConfigCommon: k8s.JobConfigCommon{
				JobName:            buildUninstallJobName(vz.Name),
				Namespace:          vz.Namespace,
				Labels:             vz.Labels,
				ServiceAccountName: buildServiceAccountName(vz.Name),
				JobImage:           os.Getenv("VZ_INSTALL_IMAGE"),
				DryRun:             r.DryRun,
			},
		},
	)

	// Set verrazzano resource as the owner and controller of the uninstall job resource.
	if err := controllerutil.SetControllerReference(vz, job, r.Scheme); err != nil {
		return err
	}

	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Infof("Checking if uninstall job %s exist", buildUninstallJobName(vz.Name))
	err := r.Get(context.TODO(), types.NamespacedName{Name: buildUninstallJobName(vz.Name), Namespace: vz.Namespace}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating uninstall job %s, dry-run=%v", buildUninstallJobName(vz.Name), r.DryRun)
		err = r.Create(context.TODO(), job)
		if err != nil {
			return err
		}

		err = r.setUninstallCondition(log, jobFound, vz)
		if err != nil {
			return err
		}

		// Job created successfully and status set successfully
		return nil
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

// buildClusterRoleBindingName returns the clusterrolebinding name for jobs based on verrazzano resource name.
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
func (r *VerrazzanoReconciler) updateStatus(log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType) error {
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
	switch conditionType {
	case installv1alpha1.InstallStarted:
		cr.Status.State = installv1alpha1.Installing
	case installv1alpha1.UninstallStarted:
		cr.Status.State = installv1alpha1.Uninstalling
	case installv1alpha1.UpgradeStarted:
		cr.Status.State = installv1alpha1.Upgrading
	case installv1alpha1.InstallComplete, installv1alpha1.UninstallComplete, installv1alpha1.UpgradeComplete:
		cr.Status.State = installv1alpha1.Ready
	case installv1alpha1.InstallFailed, installv1alpha1.UpgradeFailed, installv1alpha1.UninstallFailed:
		cr.Status.State = installv1alpha1.Failed
	}
	log.Infof("Setting verrazzano resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Error(err, "Failed to update verrazzano resource status")
		return err
	}
	return nil
}

// setInstallCondition sets the verrazzano resource condition in status for install
func (r *VerrazzanoReconciler) setInstallCondition(log *zap.SugaredLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
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
			conditionType = installv1alpha1.InstallComplete
		} else {
			message = "Verrazzano install failed to complete"
			conditionType = installv1alpha1.InstallFailed
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

// setUninstallCondition sets the verrazzano resource condition in status for uninstall
func (r *VerrazzanoReconciler) setUninstallCondition(log *zap.SugaredLogger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
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
			log.Error(err, "Failed to update uninstall job owner references")
			return err
		}

		var message string
		var conditionType installv1alpha1.ConditionType
		if job.Status.Succeeded == 1 {
			message = "Verrazzano uninstall completed successfully"
			conditionType = installv1alpha1.UninstallComplete
		} else {
			message = "Verrazzano uninstall failed to complete"
			conditionType = installv1alpha1.UninstallFailed
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
func (r *VerrazzanoReconciler) saveVerrazzanoSpec(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (err error) {
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
				Namespace: vz.Namespace,
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
func (r *VerrazzanoReconciler) getSavedInstallSpec(ctx context.Context, log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano) (*installv1alpha1.VerrazzanoSpec, error) {
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
func (r *VerrazzanoReconciler) getInternalConfigMap(ctx context.Context, vz *installv1alpha1.Verrazzano) (installConfig *corev1.ConfigMap, err error) {
	key := client.ObjectKey{
		Namespace: vz.Namespace,
		Name:      buildInternalConfigMapName(vz.Name),
	}
	installConfig = &corev1.ConfigMap{}
	err = r.Get(ctx, key, installConfig)
	return installConfig, err
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
