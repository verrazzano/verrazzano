// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vztime "github.com/verrazzano/verrazzano/pkg/time"
	"k8s.io/client-go/util/workqueue"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"strings"

	"github.com/go-logr/logr"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler constants
const (
	loggingNamePart           = "logging-stdout"
	errLoggingResource        = "cannot add logging sidecar to the resource"
	configMapAPIVersion       = "v1"
	configMapKind             = "ConfigMap"
	loggingMountPath          = "/fluentd/etc/custom.conf"
	loggingKey                = "custom.conf"
	defaultMode         int32 = 400
)

// LoggingTraitReconciler reconciles a LoggingTrait object
type LoggingTraitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingtraits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=loggingtraits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads,verbs=get;list;
// +kubebuilder:rbac:groups=core.oam.dev,resources=containerizedworkloads/status,verbs=get;
// +kubebuilder:rbac:groups=core.oam.dev,resources=workloaddefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=pods,verbs=get;list;watch;update;patch;delete

func (r *LoggingTraitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var err error
	ctx := context.Background()
	log := r.Log.WithValues("loggingtrait", req.NamespacedName)

	var trait *oamv1alpha1.LoggingTrait
	if trait, err = r.fetchTrait(ctx, req.NamespacedName); err != nil || trait == nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if trait.DeletionTimestamp.IsZero() {
		result, supported, err := r.reconcileTraitCreateOrUpdate(ctx, log, trait)
		if err != nil {
			return result, err
		}
		if !supported {
			// If the workload kind is not supported then delete the trait
			r.Log.V(1).Info(fmt.Sprintf("deleting trait %s because workload is not supported", trait.Name))

			err = r.Client.Delete(context.TODO(), trait, &client.DeleteOptions{})

		}
		return result, err
	}

	return r.reconcileTraitDelete(ctx, log, trait)
}

// reconcileTraitDelete reconciles a logging trait that is being deleted.
func (r *LoggingTraitReconciler) reconcileTraitDelete(ctx context.Context, log logr.Logger, trait *oamv1alpha1.LoggingTrait) (ctrl.Result, error) {
	// Retrieve the workload the trait is related to
	workload, err := vznav.FetchWorkloadFromTrait(ctx, r, log, trait)
	if err != nil || workload == nil {
		return reconcile.Result{}, err
	}
	if workload.GetKind() == "VerrazzanoCoherenceWorkload" || workload.GetKind() == "VerrazzanoWebLogicWorkload" {
		return reconcile.Result{}, nil
	}

	// Retrieve the child resources of the workload
	resources, err := vznav.FetchWorkloadChildren(ctx, r, log, workload)
	if err != nil {
		log.Error(err, "Error retrieving the workloads child resources", "workload", workload.UnstructuredContent())
	}

	// If there are no child resources fallback to the workload
	if len(resources) == 0 {
		resources = append(resources, workload)
	}

	for _, resource := range resources {
		isCombined := false
		configMapName := loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind())

		if ok, containersFieldPath := locateContainersField(resource); ok {
			resourceContainers, ok, err := unstructured.NestedSlice(resource.Object, containersFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource containers")
				return reconcile.Result{}, err
			}

			var image string
			if len(trait.Spec.LoggingImage) != 0 {
				image = trait.Spec.LoggingImage
			} else {
				image = os.Getenv("DEFAULT_FLUENTD_IMAGE")
			}
			envFluentd := &corev1.EnvVar{
				Name:  "FLUENTD_CONF",
				Value: "custom.conf",
			}
			loggingContainer := &corev1.Container{
				Name:            loggingNamePart,
				Image:           image,
				ImagePullPolicy: corev1.PullPolicy(trait.Spec.ImagePullPolicy),
				Env:             []corev1.EnvVar{*envFluentd},
			}

			repeatNo := 0
			repeat := false
			for i, resContainer := range resourceContainers {
				if loggingContainer.Name == resContainer.(map[string]interface{})["name"] {
					repeat = true
					repeatNo = i
					break
				}
			}
			if repeat {
				resourceContainers[repeatNo] = resourceContainers[len(resourceContainers)-1]
				resourceContainers = resourceContainers[:len(resourceContainers)-1]
			}
			err = unstructured.SetNestedSlice(resource.Object, resourceContainers, containersFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set resource containers")
				return reconcile.Result{}, err
			}

			isCombined = true

		}

		if ok, volumesFieldPath := locateVolumesField(resource); ok {
			resourceVolumes, ok, err := unstructured.NestedSlice(resource.Object, volumesFieldPath...)
			if err != nil {
				log.Error(err, "Failed to gather resource volumes")
				return reconcile.Result{}, err
			} else if !ok {
				log.Info("No volumes found")
			}

			loggingVolume := &corev1.Volume{
				Name: configMapName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
						DefaultMode: func(mode int32) *int32 {
							return &mode
						}(defaultMode),
					},
				},
			}

			repeatNo := 0
			repeat := false
			for i, resVolume := range resourceVolumes {
				if loggingVolume.Name == resVolume.(map[string]interface{})["name"] {
					log.Info("Volume was discarded because of duplicate names", "volume name", loggingVolume.Name)
					repeat = true
					repeatNo = i
					break
				}
			}
			if repeat {
				resourceVolumes[repeatNo] = resourceVolumes[len(resourceVolumes)-1]
				resourceVolumes = resourceVolumes[:len(resourceVolumes)-1]
			}

			err = unstructured.SetNestedSlice(resource.Object, resourceVolumes, volumesFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set resource containers")
				return reconcile.Result{}, err
			}

			isCombined = true

		}

		if isCombined {
			// make a copy of the resource spec since resource.Object will get overwritten in CreateOrUpdate
			// if the resource exists
			specCopy, _, err := unstructured.NestedFieldCopy(resource.Object, "spec")
			if err != nil {
				log.Error(err, "Unable to make a copy of the spec")
				return reconcile.Result{}, err
			}

			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, resource, func() error {
				return unstructured.SetNestedField(resource.Object, specCopy, "spec")
			})
			if err != nil {
				log.Error(err, "Error creating or updating resource")
				return reconcile.Result{}, err
			}
			log.Info("Successfully removed logging from resource", "resource GVK", resource.GroupVersionKind().String())
		}

		r.deleteLoggingConfigMap(ctx, trait, resource)

	}

	return reconcile.Result{}, nil
}

// fetchTrait attempts to get a trait given a namespaced name.
// Will return nil for the trait and no error if the trait does not exist.
func (r *LoggingTraitReconciler) fetchTrait(ctx context.Context, name types.NamespacedName) (*oamv1alpha1.LoggingTrait, error) {
	var trait oamv1alpha1.LoggingTrait
	r.Log.Info("Fetch trait", "trait", name)
	if err := r.Get(ctx, name, &trait); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("Trait has been deleted")
			return nil, nil
		}
		r.Log.Info("Failed to fetch trait")
		return nil, err
	}
	return &trait, nil
}

func (r *LoggingTraitReconciler) reconcileTraitCreateOrUpdate(
	ctx context.Context, log logr.Logger, trait *oamv1alpha1.LoggingTrait) (
	ctrl.Result, bool, error) {

	// Retrieve the workload the trait is related to
	workload, err := vznav.FetchWorkloadFromTrait(ctx, r, log, trait)
	if err != nil || workload == nil {
		return reconcile.Result{}, true, err
	}
	if workload.GetKind() == "VerrazzanoCoherenceWorkload" || workload.GetKind() == "VerrazzanoWebLogicWorkload" {
		return reconcile.Result{}, true, nil
	}
	// Retrieve the child resources of the workload
	resources, err := vznav.FetchWorkloadChildren(ctx, r, log, workload)
	if err != nil {
		log.Error(err, "Error retrieving the workloads child resources", "workload", workload.UnstructuredContent())
	}

	// If there are no child resources fallback to the workload
	if len(resources) == 0 {
		resources = append(resources, workload)
	}

	isFound := false
	for _, resource := range resources {
		isCombined := false
		configMapName := loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind())

		if ok, containersFieldPath := locateContainersField(resource); ok {
			resourceContainers, ok, err := unstructured.NestedSlice(resource.Object, containersFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource containers")
				return reconcile.Result{}, true, err
			}
			loggingVolumeMount := &corev1.VolumeMount{
				MountPath: loggingMountPath,
				Name:      configMapName,
				SubPath:   loggingKey,
				ReadOnly:  true,
			}
			uLoggingVolumeMount, err := struct2Unmarshal(loggingVolumeMount)
			if err != nil {
				log.Error(err, "Failed to unmarshal a volumeMount for logging")
			}

			var volumeMountFieldPath = []string{"volumeMounts"}
			var resourceVolumeMounts []interface{}
			for _, resContainer := range resourceContainers {
				volumeMounts, ok, err := unstructured.NestedSlice(resContainer.(map[string]interface{}), volumeMountFieldPath...)
				if err != nil {
					log.Error(err, "Failed to gather resource container volumeMounts")
					return reconcile.Result{}, true, err
				} else if !ok {
					log.Info("No volumeMounts found")
				}
				resourceVolumeMounts = appendSliceOfInterface(resourceVolumeMounts, volumeMounts)

			}
			iVolumeMount := -1
			for i, cVolumeMount := range resourceVolumeMounts {
				if cVolumeMount.(map[string]interface{})["mountPath"] == uLoggingVolumeMount.Object["mountPath"] {
					iVolumeMount = i
				}
			}
			if iVolumeMount == -1 {
				resourceVolumeMounts = append(resourceVolumeMounts, uLoggingVolumeMount.Object)
			}
			envFluentd := &corev1.EnvVar{
				Name:  "FLUENTD_CONF",
				Value: "custom.conf",
			}
			loggingContainer := &corev1.Container{
				Name:            loggingNamePart,
				Image:           trait.Spec.LoggingImage,
				ImagePullPolicy: corev1.PullPolicy(trait.Spec.ImagePullPolicy),
				Env:             []corev1.EnvVar{*envFluentd},
			}

			uLoggingContainer, err := struct2Unmarshal(loggingContainer)
			if err != nil {
				log.Error(err, "Failed to unmarshal a container for logging")
			}

			err = unstructured.SetNestedSlice(uLoggingContainer.Object, resourceVolumeMounts, volumeMountFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set container volumeMounts")
				return reconcile.Result{}, true, err
			}

			repeatNo := 0
			repeat := false
			for i, resContainer := range resourceContainers {
				if loggingContainer.Name == resContainer.(map[string]interface{})["name"] {
					repeat = true
					repeatNo = i
					break
				}
			}
			if repeat {
				resourceContainers[repeatNo] = uLoggingContainer.Object
			} else {
				resourceContainers = append(resourceContainers, uLoggingContainer.Object)
			}

			err = unstructured.SetNestedSlice(resource.Object, resourceContainers, containersFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set resource containers")
				return reconcile.Result{}, true, err
			}

			isCombined = true
			isFound = true

		}

		if ok, volumesFieldPath := locateVolumesField(resource); ok {
			resourceVolumes, ok, err := unstructured.NestedSlice(resource.Object, volumesFieldPath...)
			if err != nil {
				log.Error(err, "Failed to gather resource volumes")
				return reconcile.Result{}, true, err
			} else if !ok {
				log.Info("No volumes found")
			}

			loggingVolume := &corev1.Volume{
				Name: configMapName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
						DefaultMode: func(mode int32) *int32 {
							return &mode
						}(defaultMode),
					},
				},
			}
			uLoggingVolume, err := struct2Unmarshal(loggingVolume)
			if err != nil {
				log.Error(err, "Error unmarshalling logging volume")
			}

			repeatNo := 0
			repeat := false
			for i, resVolume := range resourceVolumes {
				if loggingVolume.Name == resVolume.(map[string]interface{})["name"] {
					log.Info("Volume was discarded because of duplicate names", "volume name", loggingVolume.Name)
					repeat = true
					repeatNo = i
					break
				}
			}
			if repeat {
				resourceVolumes[repeatNo] = uLoggingVolume.Object
			} else {
				resourceVolumes = append(resourceVolumes, uLoggingVolume.Object)
			}

			err = unstructured.SetNestedSlice(resource.Object, resourceVolumes, volumesFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set resource volumes")
				return reconcile.Result{}, true, err
			}

			isFound = true
			isCombined = true

		}

		if isCombined {
			if isFound {

				r.ensureLoggingConfigMapExists(ctx, trait, resource)
			}
			// make a copy of the resource spec since resource.Object will get overwritten in CreateOrUpdate
			// if the resource exists
			specCopy, _, err := unstructured.NestedFieldCopy(resource.Object, "spec")
			if err != nil {
				log.Error(err, "Unable to make a copy of the spec")
				r.deleteLoggingConfigMap(ctx, trait, resource)
				return reconcile.Result{}, true, err
			}

			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, resource, func() error {
				return unstructured.SetNestedField(resource.Object, specCopy, "spec")
			})
			if err != nil {
				log.Error(err, "Error creating or updating resource")
				r.deleteLoggingConfigMap(ctx, trait, resource)
				return reconcile.Result{}, true, err
			}
			log.Info("Successfully deploy logging to resource", "resource GVK", resource.GroupVersionKind().String())
		}

		if !isFound {
			log.Info("Cannot locate any resource", "total resources", len(resources))
			return reconcile.Result{}, false, fmt.Errorf(errLoggingResource)
		}

	}

	return reconcile.Result{}, true, nil
}

// ensureLoggingConfigMapExists ensures that the FLUENTD configmap exists. If it already exists, there is nothing
// to do. If it doesn't exist, create it.
func (r *LoggingTraitReconciler) ensureLoggingConfigMapExists(ctx context.Context, trait *oamv1alpha1.LoggingTrait, resource *unstructured.Unstructured) error {
	// check if configmap exists
	configMapName := loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind())
	configMapExists, err := resourceExists(ctx, r, configMapAPIVersion, configMapKind, configMapName, resource.GetNamespace())
	if err != nil {
		return err
	}

	if !configMapExists {
		if err = r.Create(ctx, r.createLoggingConfigMap(trait, resource), &client.CreateOptions{}); err != nil {
			return err
		}
	}
	return err
}

// createLoggingConfigMap returns a configmap based on the logging trait
func (r *LoggingTraitReconciler) createLoggingConfigMap(trait *oamv1alpha1.LoggingTrait, resource *unstructured.Unstructured) *corev1.ConfigMap {
	configMapName := loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind())
	data := make(map[string]string)
	data["custom.conf"] = trait.Spec.LoggingConfig
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: resource.GetNamespace(),
			Labels:    resource.GetLabels(),
		},
		Data: data,
	}
	controllerutil.SetControllerReference(resource, configMap, r.Scheme)
	return configMap
}

func (r *LoggingTraitReconciler) deleteLoggingConfigMap(ctx context.Context, trait *oamv1alpha1.LoggingTrait, resource *unstructured.Unstructured) error {
	// check if configmap exists
	configMapExists, err := resourceExists(ctx, r, configMapAPIVersion, configMapKind, loggingNamePart+"-"+resource.GetName()+"-"+strings.ToLower(resource.GetKind()), resource.GetNamespace())
	if configMapExists {
		return r.Delete(ctx, r.createLoggingConfigMap(trait, resource), &client.DeleteOptions{})
	}
	return err
}

// resourceExists determines whether or not a resource of the given kind identified by the given name and namespace exists
func resourceExists(ctx context.Context, r client.Reader, apiVersion string, kind string, name string, namespace string) (bool, error) {
	resources := unstructured.UnstructuredList{}
	resources.SetAPIVersion(apiVersion)
	resources.SetKind(kind)
	err := r.List(ctx, &resources, client.InNamespace(namespace), client.MatchingFields{"metadata.name": name})
	return len(resources.Items) != 0, err
}

func (r *LoggingTraitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(
				vztime.SecsToDuration(vzconst.ControllerBaseDelay),
				vztime.SecsToDuration(vzconst.ControllerMaxDelay)),
		}).
		For(&oamv1alpha1.LoggingTrait{}).
		Complete(r)
}
