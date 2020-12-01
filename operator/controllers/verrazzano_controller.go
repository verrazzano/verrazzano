// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
	"time"

	"github.com/go-logr/logr"
	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/internal/installjob"
	"github.com/verrazzano/verrazzano/operator/internal/uninstalljob"
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
)

// VerrazzanoReconciler reconciles a Verrazzano object
type VerrazzanoReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Controller controller.Controller
}

// Name of finializer
const finalizerName = "install.verrazzano.io"

// Reconcile will reconcile the CR
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.verrazzano.io,resources=verrazzanos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;watch;list;create;update;delete
func (r *VerrazzanoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := r.Log.WithValues("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name)).WithName("reconcile")

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
					log.Info(fmt.Sprintf("Removing finalizer %s", finalizerName))
					vz.ObjectMeta.Finalizers = removeString(vz.ObjectMeta.Finalizers, finalizerName)
					if err := r.Update(ctx, vz); err != nil {
						return reconcile.Result{}, err
					}
				}
			}
		}
		return reconcile.Result{}, nil
	}

	// If the version is specified then see if upgrade is needed
	if len(vz.Spec.Version) > 0 {
		if vz.Spec.Version != vz.Status.Version {
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

	err := r.createConfigMap(ctx, log, vz)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err := r.createInstallJob(ctx, log, vz, getConfigMapName(vz.Name)); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, err
}

// createServiceAccount creates a required service account
func (r *VerrazzanoReconciler) createServiceAccount(ctx context.Context, log logr.Logger, vz *installv1alpha1.Verrazzano) error {
	// Define a new service account resource
	serviceAccount := installjob.NewServiceAccount(vz.Namespace, getServiceAccountName(vz.Name), os.Getenv("IMAGE_PULL_SECRET"), vz.Labels)

	// Set verrazzano resource as the owner and controller of the service account resource.
	// This reference will result in the service account resource being deleted when the verrazzano CR is deleted.
	if err := controllerutil.SetControllerReference(vz, serviceAccount, r.Scheme); err != nil {
		return err
	}

	// Check if the service account for running the scripts exist
	serviceAccountFound := &corev1.ServiceAccount{}
	log.Info(fmt.Sprintf("Checking if install service account %s exist", getServiceAccountName(vz.Name)))
	err := r.Get(ctx, types.NamespacedName{Name: getServiceAccountName(vz.Name), Namespace: vz.Namespace}, serviceAccountFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating install service account %s", getServiceAccountName(vz.Name)))
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
func (r *VerrazzanoReconciler) createClusterRoleBinding(ctx context.Context, log logr.Logger, vz *installv1alpha1.Verrazzano) error {
	// Define a new cluster role binding resource
	clusterRoleBinding := installjob.NewClusterRoleBinding(vz, getClusterRoleBindingName(vz.Namespace, vz.Name), getServiceAccountName(vz.Name))

	// Check if the cluster role binding for running the install scripts exist
	clusterRoleBindingFound := &rbacv1.ClusterRoleBinding{}
	log.Info(fmt.Sprintf("Checking if install cluster role binding %s exist", clusterRoleBinding.Name))
	err := r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: clusterRoleBinding.Namespace}, clusterRoleBindingFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating install cluster role binding %s", clusterRoleBinding.Name))
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
func (r *VerrazzanoReconciler) createConfigMap(ctx context.Context, log logr.Logger, vz *installv1alpha1.Verrazzano) error {
	// Create the configmap resource that will contain installation configuration options
	configMap := installjob.NewConfigMap(vz.Namespace, getConfigMapName(vz.Name), vz.Labels)

	// Set the verrazzano resource as the owner and controller of the configmap
	err := controllerutil.SetControllerReference(vz, configMap, r.Scheme)
	if err != nil {
		return err
	}

	// Check if the ConfigMap exists for running the install
	configMapFound := &corev1.ConfigMap{}
	log.Info(fmt.Sprintf("Checking if install ConfigMap %s exist", configMap.Name))

	var dnsAuth *installjob.DNSAuth
	err = r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, configMapFound)
	if err != nil && errors.IsNotFound(err) {
		// Convert to json and insert into the configmap.
		dnsAuth, err = getDNSAuth(r, vz.Spec.DNS, vz.Namespace)
		if err != nil {
			return err
		}
		config, err := installjob.GetInstallConfig(vz, dnsAuth)
		if err != nil {
			return err
		}
		jsonEncoding, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return err
		}
		if dnsAuth != nil {
			configMap.Data = map[string]string{"config.json": string(jsonEncoding), installv1alpha1.OciPrivateKeyFileName: dnsAuth.PrivateKeyAuth.Key}
		} else {
			configMap.Data = map[string]string{"config.json": string(jsonEncoding)}
		}

		log.Info(fmt.Sprintf("Creating install ConfigMap %s", configMap.Name))
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
func (r *VerrazzanoReconciler) createInstallJob(ctx context.Context, log logr.Logger, vz *installv1alpha1.Verrazzano, configMapName string) error {
	// Define a new install job resource
	job := installjob.NewJob(vz.Namespace, getInstallJobName(vz.Name), vz.Labels, configMapName, getServiceAccountName(vz.Name), os.Getenv("VZ_INSTALL_IMAGE"))

	// Set verrazzano resource as the owner and controller of the job resource.
	// This reference will result in the job resource being deleted when the verrazzano CR is deleted.
	if err := controllerutil.SetControllerReference(vz, job, r.Scheme); err != nil {
		return err
	}
	// Check if the job for running the install scripts exist
	jobFound := &batchv1.Job{}
	log.Info(fmt.Sprintf("Checking if install job %s exist", getInstallJobName(vz.Name)))
	err := r.Get(ctx, types.NamespacedName{Name: getInstallJobName(vz.Name), Namespace: vz.Namespace}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating install job %s", getInstallJobName(vz.Name)))
		err = r.Create(ctx, job)
		if err != nil {
			return err
		}

		// Add our finalizer if not already added
		if !containsString(vz.ObjectMeta.Finalizers, finalizerName) {
			log.Info(fmt.Sprintf("Adding finalizer %s", finalizerName))
			vz.ObjectMeta.Finalizers = append(vz.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, vz); err != nil {
				return err
			}
		}

		// Delete leftover uninstall job if we find one.
		err = r.cleanupUninstallJob(getUninstallJobName(vz.Name), vz.Namespace, log)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	err = r.setInstallCondition(log, jobFound, vz)

	return err
}

func getDNSAuth(r *VerrazzanoReconciler, dns installv1alpha1.DNS, namespace string) (*installjob.DNSAuth, error) {
	if dns.OCI != (installv1alpha1.OCI{}) {
		secret := &corev1.Secret{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: dns.OCI.OCIConfigSecret, Namespace: namespace}, secret)
		if err != nil {
			return nil, err
		}
		dnsAuth := installjob.DNSAuth{}

		err = yaml.Unmarshal(secret.Data[installv1alpha1.OciConfigSecretFile], &dnsAuth)
		if err != nil {
			return nil, err
		}
		return &dnsAuth, nil
	}
	return nil, nil
}

// cleanupUninstallJob checks for the existence of a stale uninstall job and deletes the job if one is found
func (r *VerrazzanoReconciler) cleanupUninstallJob(jobName string, namespace string, log logr.Logger) error {
	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Info(fmt.Sprintf("Checking if stale uninstall job %s exists", jobName))
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: namespace}, jobFound)
	if err == nil {
		log.Info(fmt.Sprintf("Deleting stale uninstall job %s", jobName))
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

func (r *VerrazzanoReconciler) createUninstallJob(log logr.Logger, vz *installv1alpha1.Verrazzano) error {
	// Define a new uninstall job resource
	job := uninstalljob.NewJob(vz.Namespace, getUninstallJobName(vz.Name), vz.Labels, getServiceAccountName(vz.Name), os.Getenv("VZ_INSTALL_IMAGE"))

	// Set verrazzano resource as the owner and controller of the uninstall job resource.
	if err := controllerutil.SetControllerReference(vz, job, r.Scheme); err != nil {
		return err
	}

	// Check if the job for running the uninstall scripts exist
	jobFound := &batchv1.Job{}
	log.Info(fmt.Sprintf("Checking if uninstall job %s exist", getUninstallJobName(vz.Name)))
	err := r.Get(context.TODO(), types.NamespacedName{Name: getUninstallJobName(vz.Name), Namespace: vz.Namespace}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating uninstall job %s", getUninstallJobName(vz.Name)))
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

// getInstallJobName returns the name of an install job based on verrazzano resource name.
func getInstallJobName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// getUninstallJobName returns the name of an uninstall job based on verrazzano resource name.
func getUninstallJobName(name string) string {
	return fmt.Sprintf("verrazzano-uninstall-%s", name)
}

// getServiceAccountName returns the service account name for jobs based on verrazzano resource name.
func getServiceAccountName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// getClusterRoleBindingName returns the clusterrolebinding name for jobs based on verrazzano resource name.
func getClusterRoleBindingName(namespace string, name string) string {
	return fmt.Sprintf("verrazzano-install-%s-%s", namespace, name)
}

// getConfigMapName returns the name of a config map for an install job based on verrazzano resource name.
func getConfigMapName(name string) string {
	return fmt.Sprintf("verrazzano-install-%s", name)
}

// updateStatus updates the status in the verrazzano CR
func (r *VerrazzanoReconciler) updateStatus(log logr.Logger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType) error {
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
	log.Info(fmt.Sprintf("Setting verrazzano resource condition type/status: %v/%v", condition.Type, condition.Status))

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Error(err, "Failed to update verrazzano resource status")
		return err
	}
	return nil
}

// setInstallCondition sets the verrazzano resource condition in status for install
func (r *VerrazzanoReconciler) setInstallCondition(log logr.Logger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
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
func (r *VerrazzanoReconciler) setUninstallCondition(log logr.Logger, job *batchv1.Job, vz *installv1alpha1.Verrazzano) (err error) {
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
