// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logging

import (
	"context"
	"fmt"
	"os"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FluentdStdoutSidecarName = "fluentd-stdout-sidecar"
	fluentdConfKey           = "fluentd.conf"
	fluentdConfMountPath     = "/fluentd/etc/fluentd.conf"
	configMapName            = "fluentd-config"
	confVolume               = "fluentd-config-volume"

	scratchVolMountPath = "/scratch"
)

// DefaultFluentdImage holds the default FLUENTD image that will be used if it is not specified in the logging logInfo
var DefaultFluentdImage string

func init() {
	DefaultFluentdImage = os.Getenv("DEFAULT_FLUENTD_IMAGE")
}

// FluentdManager is a general interface to interact with FLUENTD related resources
type FluentdManager interface {
	Apply(logInfo *LogInfo, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) error
	Remove(logInfo *LogInfo, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) bool
}

// Fluentd is an implementation of FluentdManager.
type Fluentd struct {
	k8sclient.Client
	Log                    *zap.SugaredLogger
	Context                context.Context
	ParseRules             string
	StorageVolumeName      string
	StorageVolumeMountPath string
	WorkloadType           string
}

// FluentdPod contains pod information for pods which require FLUENTD integration
type FluentdPod struct {
	Containers   []corev1.Container
	Volumes      []corev1.Volume
	VolumeMounts []corev1.VolumeMount
	HandlerEnv   []corev1.EnvVar
	LogPath      string
}

// Apply applies FLUENTD configuration to create/update FLUENTD container, configmap, volumes and volume mounts.
// Returns true if any changes are made; false otherwise.
func (f *Fluentd) Apply(logInfo *LogInfo, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) error {
	if err := f.ensureFluentdConfigMapExists(resource.Namespace); err != nil {
		return err
	}

	f.ensureFluentdVolumes(fluentdPod)
	f.ensureFluentdVolumeMountExists(fluentdPod)
	f.ensureFluentdContainer(fluentdPod, logInfo, resource.Namespace)

	return nil
}

// Remove removes FLUENTD container, configmap, volumes and volume mounts.
// Returns whether the remove action has been verified so that the caller knows when it is safe to forget the association.
func (f *Fluentd) Remove(logInfo *LogInfo, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) bool {
	configMapVerified := f.removeFluentdConfigMap(resource.Namespace)
	volumesVerified := f.removeFluentdVolumes(fluentdPod)
	mountsVerified := f.removeFluentdVolumeMounts(fluentdPod)
	containersVerified := f.removeFluentdContainer(fluentdPod)

	return configMapVerified && volumesVerified && mountsVerified && containersVerified
}

// ensureFluentdContainer ensures that the FLUENTD container is in the expected state. If a FLUENTD container already
// exists, replace it with a container created with the current logInfo information. If no FLUENTD container already
// exists, create one and add it to the FluentdPod.
func (f *Fluentd) ensureFluentdContainer(fluentdPod *FluentdPod, logInfo *LogInfo, namespace string) {
	containers := fluentdPod.Containers
	fluentdContainerIndex := -1
	// iterate over existing containers looking for FLUENTD container
	for i, container := range containers {
		if container.Name == FluentdStdoutSidecarName {
			// FLUENTD container found, save the index
			fluentdContainerIndex = i
			break
		}
	}
	fluentdContainer := f.createFluentdContainer(fluentdPod, logInfo, namespace)
	if fluentdContainerIndex != -1 {
		// the index is still the initial -1 so we didn't find an existing FLUENTD container so we replace it
		containers[fluentdContainerIndex] = fluentdContainer
	} else {
		// no existing FLUENTD container was found so add it to the list
		containers = append(containers, fluentdContainer)
	}
	fluentdPod.Containers = containers
}

// ensureFluentdVolumes ensures that the FLUENTD volumes exist. We expect 2 volumes, a FLUENTD volume and a
// FLUENTD config map volume. If these already exist, nothing needs to be done. If they don't already exist,
// create them and add to the FluentdPod.
func (f *Fluentd) ensureFluentdVolumes(fluentdPod *FluentdPod) {
	volumes := fluentdPod.Volumes
	configMapVolumeExists := false
	fluentdVolumeExists := false
	for _, volume := range volumes {
		if volume.Name == f.StorageVolumeName {
			fluentdVolumeExists = true
		} else if volume.Name == fmt.Sprintf("%s-volume", configMapName) {
			configMapVolumeExists = true
		}
	}
	if !configMapVolumeExists {
		volumes = append(volumes, f.createFluentdConfigMapVolume(configMapName))
	}
	if !fluentdVolumeExists {
		volumes = append(volumes, f.createFluentdEmptyDirVolume())
	}
	fluentdPod.Volumes = volumes
}

// ensureFluentdVolumeMountExists ensures that the FLUENTD volume mount exists. If one already exists, nothing
// needs to be done. If it doesn't already exist create one and add it to the FluentdPod.
func (f *Fluentd) ensureFluentdVolumeMountExists(fluentdPod *FluentdPod) {
	volumeMounts := fluentdPod.VolumeMounts
	storageVolumeMountExists := false
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == f.StorageVolumeName {
			storageVolumeMountExists = true
		}
	}

	// If no storage volume mount exists create one and add it to the list.
	if !storageVolumeMountExists {
		volumeMounts = append(volumeMounts, f.createStorageVolumeMount())
	}

	fluentdPod.VolumeMounts = volumeMounts
}

// ensureFluentdConfigMapExists ensures that the FLUENTD configmap exists. If it already exists, there is nothing
// to do. If it doesn't exist, create it.
func (f *Fluentd) ensureFluentdConfigMapExists(namespace string) error {
	// check if configmap exists
	configMapExists, err := resourceExists(f.Context, f, configMapAPIVersion, configMapKind, configMapName+"-"+f.WorkloadType, namespace)
	if err != nil {
		return err
	}

	if configMapExists {
		return f.Update(f.Context, f.createFluentdConfigMap(namespace), &k8sclient.UpdateOptions{})
	}
	return f.Create(f.Context, f.createFluentdConfigMap(namespace), &k8sclient.CreateOptions{})
}

// createFluentdConfigMap creates the FLUENTD configmap per given namespace.
func (f *Fluentd) createFluentdConfigMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName + "-" + f.WorkloadType,
			Namespace: namespace,
		},
		Data: func() map[string]string {
			var data = make(map[string]string)
			data[fluentdConfKey] = f.ParseRules
			return data
		}(),
	}
}

// removeFluentdContainer removes FLUENTD container
func (f *Fluentd) removeFluentdContainer(fluentdPod *FluentdPod) bool {
	containers := fluentdPod.Containers
	fluentdContainerIndex := -1
	for i, container := range containers {
		if container.Name == FluentdStdoutSidecarName {
			fluentdContainerIndex = i
			break
		}
	}

	if fluentdContainerIndex >= 0 {
		length := len(containers)
		containers[fluentdContainerIndex] = containers[length-1]
		containers = containers[:length-1]
	}

	fluentdPod.Containers = containers
	// return true when we confirm that the fluentd container has been removed
	return fluentdContainerIndex == -1
}

// removeFluentdVolumeMounts removes FLUENTD volume mounts
func (f *Fluentd) removeFluentdVolumeMounts(fluentPod *FluentdPod) bool {
	// For now we can't remove the FLUENTD volume mount because we need to keep the logs in scratch
	// since we can't set 'logHomeEnabled' to false for the wls domain.
	return true
}

// removeFluentdVolumes removes FLUENTD volumes. There are currently 2 volumes, a FLUENTD volume and a
// FLUENTD configmap volume.
// Returns true if we have validated that we have already deleted the volumes; false otherwise. This ensures
// that we don't remove knowledge of the workload until we have validated that it has been fully cleaned up
// in the system.
func (f *Fluentd) removeFluentdVolumes(fluentdPod *FluentdPod) bool {
	// If the FLUENTD configmap volume exists, delete it.
	// For now we can't remove the FLUENTD volume because we need to keep the logs in scratch
	// since we can't set 'logHomeEnabled' to false for the wls domain.
	volumes := fluentdPod.Volumes
	configMapVolumeName := fmt.Sprintf("%s-volume", configMapName)
	configMapVolumeIndex := -1
	for i, volume := range volumes {
		if volume.Name == configMapVolumeName {
			configMapVolumeIndex = i
			break
		}
	}

	if configMapVolumeIndex >= 0 {
		length := len(volumes)
		volumes[configMapVolumeIndex] = volumes[length-1]
		volumes = volumes[:length-1]
	}

	fluentdPod.Volumes = volumes
	// return true when we verify that volumes have been removed
	return configMapVolumeIndex == -1
}

// removeFluentdConfigMap removes the FLUENTD configmap
func (f *Fluentd) removeFluentdConfigMap(namespace string) bool {
	configMapExists, err := resourceExists(f.Context, f, configMapAPIVersion, configMapKind, configMapName+"-"+f.WorkloadType, namespace)

	if configMapExists {
		_ = f.Delete(f.Context, f.createFluentdConfigMap(namespace), &k8sclient.DeleteOptions{})
	}
	// return true when we confirm that the configmap has been successfully deleted
	return !(configMapExists) && err == nil
}

// createFluentdContainer creates the FLUENTD stdout sidecar container
func (f *Fluentd) createFluentdContainer(fluentdPod *FluentdPod, logInfo *LogInfo, namespace string) corev1.Container {
	container := corev1.Container{
		Name:            FluentdStdoutSidecarName,
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           logInfo.FluentdImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "LOG_PATH",
				Value: fluentdPod.LogPath,
			},
			{
				Name:  "FLUENTD_CONF",
				Value: fluentdConfKey,
			},
			{
				Name:  "NAMESPACE",
				Value: namespace,
			},
			{
				Name: "APP_CONF_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['" + oam.LabelAppName + "']",
					},
				},
			},
			{
				Name: "COMPONENT_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['" + oam.LabelAppComponent + "']",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: fluentdConfMountPath,
				Name:      confVolume,
				SubPath:   fluentdConfKey,
				ReadOnly:  true,
			},
			{
				MountPath: f.StorageVolumeMountPath,
				Name:      f.StorageVolumeName,
				ReadOnly:  true,
			},
		},
	}

	// add handler specific env vars
	container.Env = append(container.Env, fluentdPod.HandlerEnv...)

	return container
}

// createFluentdEmptyDirVolume creates an empty FLUENTD directory volume
func (f *Fluentd) createFluentdEmptyDirVolume() corev1.Volume {
	return corev1.Volume{
		Name: f.StorageVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// createFluentdConfigMapVolume creates a FLUENTD configmap volume
func (f *Fluentd) createFluentdConfigMapVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: fmt.Sprintf("%s-volume", name),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name + "-" + f.WorkloadType,
				},
				DefaultMode: func(mode int32) *int32 {
					return &mode
				}(420),
			},
		},
	}
}

// createStorageVolumeMount creates a storage volume mount
func (f *Fluentd) createStorageVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      f.StorageVolumeName,
		MountPath: f.StorageVolumeMountPath,
	}
}

// resourceExists determines whether or not a resource of the given kind identified by the given name and namespace exists
func resourceExists(ctx context.Context, r k8sclient.Reader, apiVersion, kind, name, namespace string) (bool, error) {
	resources := unstructured.UnstructuredList{}
	resources.SetAPIVersion(apiVersion)
	resources.SetKind(kind)
	err := r.List(ctx, &resources, k8sclient.InNamespace(namespace), k8sclient.MatchingFields{"metadata.name": name})
	return len(resources.Items) != 0, err
}
