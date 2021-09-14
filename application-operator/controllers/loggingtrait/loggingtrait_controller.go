// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"context"
	"fmt"

	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"

	crossplanev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
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
	loggingNamePart             = "logging-stdout"
	errPatchTobeLoggingResource = "cannot patch the resource for logging container"
	errLoggingResource          = "cannot add logging sidecar to the resource"
	errQueryOpenAPI             = "failed to query openAPI"
	configMapAPIVersion         = "v1"
	configMapKind               = "ConfigMap"
	loggingMountPath            = "/fluentd/etc/fluentd.conf"
	loggingVolume               = "logging-stdout-volume"
	loggingKey                  = "fluentd.conf"
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

	// If the trait no longer exists or is being deleted then return success.
	if trait == nil || isTraitBeingDeleted(trait) {
		return reconcile.Result{}, nil
	}

	// Retrieve the workload the trait is related to
	workload, err := vznav.FetchWorkloadFromTrait(ctx, r, log, trait)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Retrieve the child resources of the workload
	resources, err := vznav.FetchWorkloadChildren(ctx, r, log, workload)
	if err != nil {
		log.Error(err, "Error retrieving the workloads child resources", "workload", workload.UnstructuredContent())
		return util.ReconcileWaitResult, util.PatchCondition(ctx, r, trait, crossplanev1alpha1.ReconcileError(fmt.Errorf(util.ErrFetchChildResources)))
	}

	// If there are no child resources fallback to the workload
	if len(resources) == 0 {
		resources = append(resources, workload)
	}

	//Apply logging trait as sidecar to resources
	result, err := r.logResource(ctx, log, trait, resources)
	if err != nil {
		log.Error(err, "Error patching logging to resources")
		return result, err
	}

	return reconcile.Result{}, nil
}

// isTraitBeingDeleted determines if the trait is in the process of being deleted.
// This is done checking for a non-nil deletion timestamp.
func isTraitBeingDeleted(trait *oamv1alpha1.LoggingTrait) bool {
	return trait != nil && trait.GetDeletionTimestamp() != nil
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

func (r *LoggingTraitReconciler) logResource(
	ctx context.Context, log logr.Logger, trait *oamv1alpha1.LoggingTrait, resources []*unstructured.Unstructured) (
	ctrl.Result, error) {

	schema, err := r.DiscoveryClient.OpenAPISchema()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errQueryOpenAPI)
	}
	document, err := openapi.NewOpenAPIData(schema)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errQueryOpenAPI)
	}
	isFound := false
	for _, resource := range resources {
		isCombined := false
		r.ensureLoggingConfigMapExists(ctx, trait, resource)

		if ok, containersFieldPath := locateContainersField(document, resource); ok {
			resourceContainers, ok, err := unstructured.NestedSlice(resource.Object, containersFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource containers")
				return reconcile.Result{}, errors.Wrap(err, errPatchTobeLoggingResource)
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

			var resVolumeMounts []*unstructured.Unstructured

			for _, resContainer := range resourceContainers {
				res := resContainer.(*unstructured.Unstructured)

				if ok, volumeMountsFieldPath := locateVolumeMountsField(document, res); ok {
					resourceVolumeMounts, ok, err := unstructured.NestedSlice(res.Object, volumeMountsFieldPath...)
					if !ok || err != nil {
						log.Error(err, "Failed to gather resource container volumeMounts")
						return reconcile.Result{}, errors.Wrap(err, errPatchTobeLoggingResource)
					}

					for _, resourceVolumeMount := range resourceVolumeMounts {
						resVolumeMounts = append(resVolumeMounts, resourceVolumeMount.(*unstructured.Unstructured))
					}
				}

			}
			resVolumeMounts = append(resVolumeMounts, &uLoggingVolumeMount)
			var loggingVolumeMounts = make(map[string]interface{})
			loggingVolumeMounts["volumeMounts"] = resVolumeMounts

			loggingContainer := &corev1.Container{
				Name:            loggingNamePart,
				Image:           trait.Spec.LoggingImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
			}

			uLoggingContainer, err := struct2Unmarshal(loggingContainer)
			if err != nil {
				log.Error(err, "Failed to unmarshal a container for logging")
			}

			uLoggingContainer.SetUnstructuredContent(loggingVolumeMounts)

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

			var containers = make(map[string]interface{})
			containers["containers"] = resourceContainers

			resource.SetUnstructuredContent(containers)

			isCombined = true
			isFound = true

		}

		if ok, volumesFieldPath := locateVolumesField(document, resource); ok {
			resourceVolumes, ok, err := unstructured.NestedSlice(resource.Object, volumesFieldPath...)
			if !ok || err != nil {
				log.Error(err, "Failed to gather resource volumes")
				return reconcile.Result{}, errors.Wrap(err, errPatchTobeLoggingResource)
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

			var volumes = make(map[string]interface{})
			volumes["volumes"] = resourceVolumes

			resource.SetUnstructuredContent(volumes)

			isFound = true
			isCombined = true

		}

		if isCombined {
			// make a copy of the resource spec since resource.Object will get overwritten in CreateOrUpdate
			// if the resource exists
			specCopy, _, err := unstructured.NestedFieldCopy(resource.Object, "spec")
			if err != nil {
				log.Error(err, "Unable to make a copy of the Coherence spec")
				return reconcile.Result{}, err
			}

			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, resource, func() error {
				return unstructured.SetNestedField(resource.Object, specCopy, "spec")
			})
			if err != nil {
				log.Error(err, "Error creating or updating resource")
				return reconcile.Result{}, err
			}
			log.Info("Successfully deploy logging to resource", "resource GVK", resource.GroupVersionKind().String())
		}

		if !isFound {
			log.Info("Cannot locate any resource", "total resources", len(resources))
			return reconcile.Result{}, fmt.Errorf(errLoggingResource)
		}

	}

	return reconcile.Result{}, nil
}

// ensureLoggingConfigMapExists ensures that the FLUENTD configmap exists. If it already exists, there is nothing
// to do. If it doesn't exist, create it.
func (r *LoggingTraitReconciler) ensureLoggingConfigMapExists(ctx context.Context, trait *oamv1alpha1.LoggingTrait, resource *unstructured.Unstructured) error {
	// check if configmap exists
	configMapExists, err := resourceExists(ctx, r, configMapAPIVersion, configMapKind, loggingNamePart+"-"+resource.GetName(), resource.GetNamespace())
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
			Name:      loggingNamePart + "-" + resource.GetName(),
			Namespace: resource.GetNamespace(),
		},
		Data: trait.Spec.LoggingConfig,
	}
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
