// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/image-patch-operator/internal/k8s"

	"github.com/verrazzano/verrazzano/image-patch-operator/controllers/imagejob"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imagesv1alpha1 "github.com/verrazzano/verrazzano/image-patch-operator/api/images/v1alpha1"
)

// ImageBuildRequestReconciler reconciles a ImageBuildRequest object
type ImageBuildRequestReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Controller controller.Controller
	DryRun     bool
}

// Name of finalizer
const finalizerName = "images.verrazzano.io"

const serviceAccountName = "verrazzano-image-build-job"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageBuildRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.4/pkg/reconcile
func (r *ImageBuildRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))

	log.Info("Reconciler called")

	ibr := &imagesv1alpha1.ImageBuildRequest{}

	if err := r.Get(ctx, req.NamespacedName, ibr); err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the ImageBuildRequest resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Error getting the ImageBuildRequest resource - don't requeue.
		log.Errorf("Failed to fetch ImageBuildRequest resource: %v", err)
		return reconcile.Result{}, err
	}

	if !ibr.ObjectMeta.DeletionTimestamp.IsZero() {
		// Cancel any running install jobs before installing
		if err := r.cancelImageJob(log, ibr); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	//// The verrazzano resource is being deleted
	//if !ibr.ObjectMeta.DeletionTimestamp.IsZero() {
	//	// Finalizer is present, so lets do the uninstall
	//	if containsString(ibr.ObjectMeta.Finalizers, finalizerName) {
	//		// Cancel any running install jobs before installing
	//		if err := r.cancelImageJob(log, ibr); err != nil {
	//			return reconcile.Result{}, err
	//		}
	//	}
	//	return reconcile.Result{}, nil
	//}

	if err := r.createImageJob(ctx, log, ibr, buildConfigMapName(ibr.Name)); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createImageJob creates the installation job
func (r *ImageBuildRequestReconciler) createImageJob(ctx context.Context, log *zap.SugaredLogger, ibr *imagesv1alpha1.ImageBuildRequest, configMapName string) error {
	// Define a new image job resource
	job := imagejob.NewJob(
		&imagejob.JobConfig{
			JobConfigCommon: k8s.JobConfigCommon{
				JobName:            buildImageJobName(ibr.Name),
				Namespace:          ibr.Namespace,
				Labels:             ibr.Labels,
				ServiceAccountName: serviceAccountName,
				JobImage:           "busybox",
				DryRun:             r.DryRun,
			},
			ConfigMapName: configMapName,
		})
	//log.Infof("getting job info: %v", job)
	// Set ImageBuildRequest resource as the owner and controller of the image job resource.
	if err := controllerutil.SetControllerReference(ibr, job, r.Scheme); err != nil {
		return err
	}
	jobFound := &batchv1.Job{}
	log.Infof("Checking if image job %s exist", buildImageJobName(ibr.Name))
	err := r.Get(ctx, types.NamespacedName{Name: buildImageJobName(ibr.Name), Namespace: ibr.Namespace}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Infof("Creating image job %s, dry-run=%v", buildImageJobName(ibr.Name), r.DryRun)
		err = r.Create(ctx, job)
		if err != nil {
			return err
		}

		// Add our finalizer if not already added
		if !containsString(ibr.ObjectMeta.Finalizers, finalizerName) {
			log.Infof("Adding finalizer %s", finalizerName)
			ibr.ObjectMeta.Finalizers = append(ibr.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, ibr); err != nil {
				return err
			}
		}

		//// Delete leftover uninstall job if we find one.
		//err = r.cleanupUninstallJob(buildUninstallJobName(vz.Name), vz.Namespace, log)
		//if err != nil {
		//	return err
		//}

	} else if err != nil {
		return err
	}

	//log.Infof("Creating image job %s, dry-run=%v, job=%v", buildImageJobName(ibr.Name), r.DryRun, job)
	//err := r.Create(ctx, job)
	//if err != nil {
	//	return err
	//}
	//
	//// Add our finalizer if not already added
	//if !containsString(ibr.ObjectMeta.Finalizers, finalizerName) {
	//	log.Infof("Adding finalizer %s", finalizerName)
	//	ibr.ObjectMeta.Finalizers = append(ibr.ObjectMeta.Finalizers, finalizerName)
	//	if err := r.Update(ctx, ibr); err != nil {
	//		return err
	//	}
	//}

	//// Set the version in the status.  This will be updated when the starting install condition is updated.
	//chartSemVer, err := imagesv1alpha1.GetCurrentChartVersion()
	//if err != nil {
	//	return err
	//}
	//ibr.Status.Version = chartSemVer.ToString()

	log.Info("about to set Image build condition")
	err = r.setImageBuildCondition(log, jobFound, ibr)
	log.Infof("error from set image build condition %v", err)

	return err
}

// cancelImageJob Cancels a running install job by deleting the batch object
func (r *ImageBuildRequestReconciler) cancelImageJob(log *zap.SugaredLogger, ibr *imagesv1alpha1.ImageBuildRequest) error {
	// Check if the job for running the install scripts exist
	jobName := buildImageJobName(ibr.Name)
	jobFound := &batchv1.Job{}
	log.Debugf("Checking if image job %s exist", jobName)
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: ibr.Namespace}, jobFound)
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
	log.Infof("Image job %s in progress, deleting", jobName)
	return r.Delete(context.TODO(), jobFound, deleteOptions)
}

// updateStatus updates the status in the verrazzano CR
func (r *ImageBuildRequestReconciler) updateStatus(log *zap.SugaredLogger, cr *imagesv1alpha1.ImageBuildRequest, message string, conditionType imagesv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := imagesv1alpha1.Condition{
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
	case imagesv1alpha1.BuildStarted:
		cr.Status.State = imagesv1alpha1.Building
	case imagesv1alpha1.BuildCompleted:
		cr.Status.State = imagesv1alpha1.Published
	case imagesv1alpha1.BuildFailed:
		cr.Status.State = imagesv1alpha1.Failed
	}
	log.Infof("Setting ImageBuildRequest resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	err := r.Status().Update(context.TODO(), cr)
	if err != nil {
		log.Errorf("Failed to update ImageBuildRequest resource status: %v", err)
		return err
	}
	return nil
}

// setImageBuildCondition sets the verrazzano resource condition in status for install
func (r *ImageBuildRequestReconciler) setImageBuildCondition(log *zap.SugaredLogger, job *batchv1.Job, ibr *imagesv1alpha1.ImageBuildRequest) (err error) {
	// If the job has succeeded or failed add the appropriate condition

	log.Info(job.Status.Succeeded)

	if job.Status.Succeeded != 0 || job.Status.Failed != 0 {
		for _, condition := range ibr.Status.Conditions {
			if condition.Type == imagesv1alpha1.BuildCompleted || condition.Type == imagesv1alpha1.BuildFailed {
				return nil
			}
		}
		var message string
		var conditionType imagesv1alpha1.ConditionType
		if job.Status.Succeeded == 1 {
			message = "ImageBuildRequest install completed successfully"
			conditionType = imagesv1alpha1.BuildCompleted
		} else {
			message = "ImageBuildRequest install failed to complete"
			conditionType = imagesv1alpha1.BuildFailed
		}
		log.Info("Trying to update status to completed or failed")
		return r.updateStatus(log, ibr, message, conditionType)
	}

	// Add the install started condition if not already added
	for _, condition := range ibr.Status.Conditions {
		if condition.Type == imagesv1alpha1.BuildStarted {
			return nil
		}
	}

	return r.updateStatus(log, ibr, "ImageBuildRequest install in progress", imagesv1alpha1.BuildStarted)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageBuildRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&imagesv1alpha1.ImageBuildRequest{}).Build(r)
	return err
}

// buildImageJobName returns the name of an image job based on verrazzano resource name.
func buildImageJobName(name string) string {
	return fmt.Sprintf("verrazzano-images-%s", name)
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

// buildConfigMapName returns the name of a config map for an install job based on verrazzano resource name.
func buildConfigMapName(name string) string {
	return fmt.Sprintf("verrazzano-images-%s", name)
}
