// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/kubectl/pkg/util/openapi"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler constants
const (
	loggingNamePart     = "logging-stdout"
	errLoggingResource  = "cannot add logging sidecar to the resource"
	errQueryOpenAPI     = "failed to query openAPI"
	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"
	loggingMountPath    = "/fluentd/etc/fluentd.conf"
	loggingVolume       = "logging-stdout-volume"
	loggingKey          = "fluentd.conf"
)

// LoggingTraitReconciler reconciles a LoggingTrait object
type LoggingTraitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Record event.Recorder
	discovery.DiscoveryClient
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

	// Retrieve the child resources of the workload
	resources, err := vznav.FetchWorkloadChildren(ctx, r, log, workload)
	if err != nil {
		log.Error(err, "Error retrieving the workloads child resources", "workload", workload.UnstructuredContent())
	}

	// If there are no child resources fallback to the workload
	if len(resources) == 0 {
		resources = append(resources, workload)
	}

	schema, err := r.DiscoveryClient.OpenAPISchema()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errQueryOpenAPI)
	}
	document, err := openapi.NewOpenAPIData(schema)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errQueryOpenAPI)
	}

	for _, resource := range resources {
		isCombined := false

		if ok, containersFieldPath := locateContainersField(document, resource); ok {
			resourceContainers, ok, err := unstructured.NestedSlice(resource.Object, containersFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource containers")
				return reconcile.Result{}, errors.Wrap(err, "Failed to gather resource containers")
			}

			var image string
			if len(trait.Spec.LoggingImage) != 0 {
				image = trait.Spec.LoggingImage
			} else {
				image = os.Getenv("DEFAULT_FLUENTD_IMAGE")
			}

			loggingContainer := &corev1.Container{
				Name:            loggingNamePart,
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
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
				return reconcile.Result{}, errors.Wrap(err, "Unable to set resource containers")
			}

			isCombined = true

		}

		if ok, volumesFieldPath := locateVolumesField(document, resource); ok {
			resourceVolumes, ok, err := unstructured.NestedSlice(resource.Object, volumesFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource volumes")
				return reconcile.Result{}, errors.Wrap(err, errLoggingResource)
			}

			loggingVolume := &corev1.Volume{
				Name: loggingVolume,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: loggingNamePart + "-" + resource.GetName(),
						},
						DefaultMode: func(mode int32) *int32 {
							return &mode
						}(420),
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
				return reconcile.Result{}, errors.Wrap(err, "Unable to set resource containers")
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

	// Retrieve the child resources of the workload
	resources, err := vznav.FetchWorkloadChildren(ctx, r, log, workload)
	if err != nil {
		log.Error(err, "Error retrieving the workloads child resources", "workload", workload.UnstructuredContent())
	}

	// If there are no child resources fallback to the workload
	if len(resources) == 0 {
		resources = append(resources, workload)
	}

	schema, err := r.DiscoveryClient.OpenAPISchema()
	if err != nil {
		return reconcile.Result{}, true, errors.Wrap(err, errQueryOpenAPI)
	}
	document, err := openapi.NewOpenAPIData(schema)
	if err != nil {
		return reconcile.Result{}, true, errors.Wrap(err, errQueryOpenAPI)
	}
	isFound := false
	for _, resource := range resources {
		isCombined := false

		if ok, containersFieldPath := locateContainersField(document, resource); ok {
			resourceContainers, ok, err := unstructured.NestedSlice(resource.Object, containersFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource containers")
				return reconcile.Result{}, true, errors.Wrap(err, errLoggingResource)
			}
			loggingVolumeMount := &corev1.VolumeMount{
				MountPath: loggingMountPath,
				Name:      loggingVolume,
				SubPath:   loggingKey,
				ReadOnly:  true,
			}
			uLoggingVolumeMount, err := struct2Unmarshal(loggingVolumeMount)
			if err != nil {
				log.Error(err, "Failed to unmarshal a volumeMount for logging")
			}

			var volumeMountFieldPath []string
			var resourceVolumeMounts []interface{}
			for _, resContainer := range resourceContainers {
				res := resContainer.(*unstructured.Unstructured)
				volumeMounts, ok, err := unstructured.NestedSlice(res.Object, []string{"volumeMounts"}...)
				if !ok || err != nil {
					log.Error(err, "Failed to gather resource container volumeMounts")
					return reconcile.Result{}, true, errors.Wrap(err, errLoggingResource)
				}
				resourceVolumeMounts = append(resourceVolumeMounts, volumeMounts...)

			}
			resVolumeMounts := append(resourceVolumeMounts, uLoggingVolumeMount.Object)

			loggingContainer := &corev1.Container{
				Name:            loggingNamePart,
				Image:           trait.Spec.LoggingImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
			}

			uLoggingContainer, err := struct2Unmarshal(loggingContainer)
			if err != nil {
				log.Error(err, "Failed to unmarshal a container for logging")
			}

			err = unstructured.SetNestedSlice(uLoggingContainer.Object, resVolumeMounts, volumeMountFieldPath...)
			if err != nil {
				log.Error(err, "Unable to set container volumeMounts")
				return reconcile.Result{}, true, errors.Wrap(err, "Unable to set container volumeMounts")
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
				return reconcile.Result{}, true, errors.Wrap(err, "Unable to set resource containers")
			}

			isCombined = true
			isFound = true

		}

		if ok, volumesFieldPath := locateVolumesField(document, resource); ok {
			resourceVolumes, ok, err := unstructured.NestedSlice(resource.Object, volumesFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource volumes")
				return reconcile.Result{}, true, errors.Wrap(err, errLoggingResource)
			}

			loggingVolume := &corev1.Volume{
				Name: loggingVolume,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind()),
						},
						DefaultMode: func(mode int32) *int32 {
							return &mode
						}(420),
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
				return reconcile.Result{}, true, errors.Wrap(err, "Unable to set resource volumes")
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
	configMapExists, err := resourceExists(ctx, r, configMapAPIVersion, configMapKind, loggingNamePart+"-"+resource.GetName()+"-"+strings.ToLower(resource.GetKind()), resource.GetNamespace())
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
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingNamePart + "-" + resource.GetName() + "-" + strings.ToLower(resource.GetKind()),
			Namespace: resource.GetNamespace(),
			Labels:    resource.GetLabels(),
		},
		Data: trait.Spec.LoggingConfig,
	}
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
		For(&oamv1alpha1.LoggingTrait{}).
		Complete(r)
}
