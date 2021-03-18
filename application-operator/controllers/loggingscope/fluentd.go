// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"os"

	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	fluentdConfKey       = "fluentd.conf"
	fluentdConfMountPath = "/fluentd/etc/fluentd.conf"
	fluentdContainerName = "fluentd"
	configMapName        = "fluentd-config"
	scratchVolMountPath  = "/scratch"

	elasticSearchURLEnv  = "ELASTICSEARCH_URL"
	elasticSearchUserEnv = "ELASTICSEARCH_USER"
	elasticSearchPwdEnv  = "ELASTICSEARCH_PASSWORD"

	secretVolume    = "secret-volume"
	secretMountPath = "/fluentd/secret"
)

// ElasticSearchIndex defines the common index pattern
const ElasticSearchIndex = "#{ENV['NAMESPACE']}-#{ENV['APP_CONF_NAME']}-#{ENV['COMPONENT_NAME']}"

const (
	// DefaultElasticSearchURL defines the default Elasticsearch URL used if it is not specified in the logging scope
	DefaultElasticSearchURL = "http://vmi-system-es-ingest.verrazzano-system.svc.cluster.local:9200"

	// DefaultSecretName defines the default Elasticsearch secret name used if it is not specified in the logging scope
	DefaultSecretName = "verrazzano"
)

// DefaultFluentdImage holds the default FLUENTD image that will be used if it is not specified in the logging scope
var DefaultFluentdImage string

func init() {
	DefaultFluentdImage = os.Getenv("DEFAULT_FLUENTD_IMAGE")
}

// FluentdManager is a general interface to interact with FLUENTD related resources
type FluentdManager interface {
	Apply(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) (bool, error)
	Remove(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) bool
}

// Fluentd is an implementation of FluentdManager.
type Fluentd struct {
	k8sclient.Client
	Log                    logr.Logger
	Context                context.Context
	ParseRules             string
	StorageVolumeName      string
	StorageVolumeMountPath string
	WorkloadType           string
}

// FluentdPod contains pod information for pods which require FLUENTD integration
type FluentdPod struct {
	Containers   []v1.Container
	Volumes      []v1.Volume
	VolumeMounts []v1.VolumeMount
	HandlerEnv   []v1.EnvVar
	LogPath      string
}

// Apply applies FLUENTD configuration to create/update FLUENTD container, configmap, volumes and volume mounts.
// Returns true if any changes are made; false otherwise.
func (f *Fluentd) Apply(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) (bool, error) {
	upToDate := f.isFluentdContainerUpToDate(fluentdPod.Containers, scope)
	if !upToDate {
		err := f.ensureFluentdConfigMapExists(resource.Namespace)
		if err != nil {
			return false, err
		}

		ensureElasticsearchSecret(f.Context, f, resource.Namespace, scope.Spec.SecretName)
		f.ensureFluentdVolumes(fluentdPod, scope)
		f.ensureFluentdVolumeMountExists(fluentdPod)
		f.ensureFluentdContainer(fluentdPod, scope, resource.Namespace)
		return true, nil
	}
	return false, nil
}

// Remove removes FLUENTD container, configmap, volumes and volume mounts.
// Returns whether the remove action has been verified so that the caller knows when it is safe to forget the association.
func (f *Fluentd) Remove(scope *vzapi.LoggingScope, resource vzapi.QualifiedResourceRelation, fluentdPod *FluentdPod) bool {
	configMapVerified := f.removeFluentdConfigMap(resource.Namespace)
	volumesVerified := f.removeFluentdVolumes(fluentdPod)
	mountsVerified := f.removeFluentdVolumeMounts(fluentdPod)
	containersVerified := f.removeFluentdContainer(fluentdPod)

	return configMapVerified && volumesVerified && mountsVerified && containersVerified
}

// ensureFluentdContainer ensures that the FLUENTD container is in the expected state. If a FLUENTD container already
// exists, replace it with a container created with the current scope information. If no FLUENTD container already
// exists, create one and add it to the FluentdPod.
func (f *Fluentd) ensureFluentdContainer(fluentdPod *FluentdPod, scope *vzapi.LoggingScope, namespace string) {
	containers := fluentdPod.Containers
	fluentdContainerIndex := -1
	// iterate over existing containers looking for FLUENTD container
	for i, container := range containers {
		if container.Name == "fluentd" {
			// FLUENTD container found, save the index
			fluentdContainerIndex = i
			break
		}
	}
	fluentdContainer := f.createFluentdContainer(fluentdPod, scope, namespace)
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
func (f *Fluentd) ensureFluentdVolumes(fluentdPod *FluentdPod, scope *vzapi.LoggingScope) {
	volumes := fluentdPod.Volumes
	configMapVolumeExists := false
	fluentdVolumeExists := false
	secretVolumeExists := false
	for _, volume := range volumes {
		if volume.Name == f.StorageVolumeName {
			fluentdVolumeExists = true
		} else if volume.Name == fmt.Sprintf("%s-volume", configMapName) {
			configMapVolumeExists = true
		} else if volume.Name == secretVolume {
			secretVolumeExists = true
		}
	}
	if !configMapVolumeExists {
		volumes = append(volumes, f.createFluentdConfigMapVolume(configMapName))
	}
	if !secretVolumeExists {
		volumes = append(volumes, f.createFluentdSecretVolume(scope.Spec.SecretName))
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
	secretVolumeMountExists := false
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == f.StorageVolumeName {
			storageVolumeMountExists = true
		} else if volumeMount.Name == secretVolume {
			secretVolumeMountExists = true
		}
	}

	// If no storage volume mount exists create one and add it to the list.
	if !storageVolumeMountExists {
		volumeMounts = append(volumeMounts, f.createStorageVolumeMount())
	}

	// If no secret volume mount exists create one and add it to the list.
	if !secretVolumeMountExists {
		volumeMounts = append(volumeMounts, f.createSecretVolumeMount())
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

	if !configMapExists {
		if err = f.Create(f.Context, f.createFluentdConfigMap(namespace), &k8sclient.CreateOptions{}); err != nil {
			return err
		}
	}
	return err
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
		if container.Name == fluentdContainerName {
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

// isFluentdContainerUpToDate is used to determine if the FLUENTD container is in the current expected state
func (f *Fluentd) isFluentdContainerUpToDate(containers []v1.Container, scope *vzapi.LoggingScope) bool {
	for _, container := range containers {
		if container.Name != fluentdContainerName {
			continue
		}
		diffExists := false
		for _, envvar := range container.Env {
			switch name := envvar.Name; name {
			case elasticSearchURLEnv:
				host := envvar.Value
				f.Log.Info("FLUENTD container ElasticSearch url", "url", host)
				if host != scope.Spec.ElasticSearchURL {
					diffExists = true
					break
				}
			case elasticSearchUserEnv:
				secretName := envvar.ValueFrom.SecretKeyRef.LocalObjectReference.Name
				f.Log.Info("FLUENTD container ElasticSearch user secret", "secret", secretName)
				if secretName != scope.Spec.SecretName {
					diffExists = true
					break
				}
			case elasticSearchPwdEnv:
				secretName := envvar.ValueFrom.SecretKeyRef.LocalObjectReference.Name
				f.Log.Info("FLUENTD container ElasticSearch password secret", "secret", secretName)
				if secretName != scope.Spec.SecretName {
					diffExists = true
					break
				}
			}
			f.Log.Info("FLUENTD container image", "image", container.Image)
			if container.Image != scope.Spec.FluentdImage {
				diffExists = true
				break
			}
		}
		return !diffExists
	}
	return false
}

// createFluentdContainer creates the FLUENTD container
func (f *Fluentd) createFluentdContainer(fluentdPod *FluentdPod, scope *vzapi.LoggingScope, namespace string) corev1.Container {
	container := corev1.Container{
		Name:            "fluentd",
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           scope.Spec.FluentdImage,
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
				Name:  "FLUENT_ELASTICSEARCH_SED_DISABLE",
				Value: "true",
			},
			{
				Name:  elasticSearchURLEnv,
				Value: scope.Spec.ElasticSearchURL,
			},
			{
				Name: elasticSearchUserEnv,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: scope.Spec.SecretName,
						},
						Key: constants.ElasticsearchUsernameData,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name: elasticSearchPwdEnv,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: scope.Spec.SecretName,
						},
						Key: constants.ElasticsearchPasswordData,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name:  "NAMESPACE",
				Value: namespace,
			},
			{
				Name: "APP_CONF_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.labels['" + oam.LabelAppName + "']",
					},
				},
			},
			{
				Name: "COMPONENT_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.labels['" + oam.LabelAppComponent + "']",
					},
				},
			},
			{
				Name:  "CLUSTER_NAME",
				Value: clusters.GetClusterName(context.TODO(), f.Client),
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
				MountPath: secretMountPath,
				Name:      secretVolume,
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

// createFluentdSecretVolume creates a FLUENTD secret volume
func (f *Fluentd) createFluentdSecretVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: secretVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName},
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

// createSecretVolumeMount creates a secret volume mount
func (f *Fluentd) createSecretVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      secretVolume,
		MountPath: secretMountPath,
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

func ensureElasticsearchSecret(ctx context.Context, cli k8sclient.Client, namespace, name string) error {
	secret := &corev1.Secret{}
	err := cli.Get(ctx, objKey(namespace, name), secret)
	if kerrs.IsNotFound(err) {
		// If this is a managed cluster, and we are using the managed cluster ES secret, copy
		// that secret to the app namespace
		if shouldUseManagedClusterElasticsearchSecret(ctx, cli, name) {
			// The managed cluster ES secret is the one specified on the logging scope - copy it
			// to the app namespace
			return copyManagedClusterVMISecret(ctx, cli, namespace, name)
		}
		// create an empty secret, which is required in order to mount the secret
		// as a volume in fluentd. In certain cases (e.g. admin server using local Elasticsearch),
		// the secret is not required to have contents. In other cases, where user explicitly
		// specifies a secret on the logging scope, they should have already created it in the app NS
		return createEmptySecretForFluentdVolume(ctx, cli, namespace, name)
	}
	return err
}

func createEmptySecretForFluentdVolume(ctx context.Context, cli k8sclient.Client, namespace string, name string) error {
	placeholderSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return cli.Create(ctx, placeholderSecret, &k8sclient.CreateOptions{})
}

// copies the managed cluster Elasticsearch secret to the given namespace/name IF it exists
func copyManagedClusterVMISecret(ctx context.Context, cli k8sclient.Client, namespace string, name string) error {
	secretKey := clusters.GetManagedClusterElasticsearchSecretKey()
	if name != secretKey.Name {
		// The managed cluster ES secret is not the one specified on the logging scope
		// nothing to copy
		return nil
	}
	secret := &corev1.Secret{}
	err := cli.Get(ctx, secretKey, secret)
	if kerrs.IsNotFound(err) {
		// Not a managed cluster, nothing to replicate
		return nil
	}
	if err != nil {
		return err
	}
	secret = replicateVmiSecret(secret, namespace, name)
	err = cli.Create(ctx, secret, &k8sclient.CreateOptions{})
	if kerrs.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// shouldUseManagedClusterElasticsearchSecret returns true if this is a managed cluster and the
// logging scope specifies that the managed cluster Elasticsearch secret should be used
func shouldUseManagedClusterElasticsearchSecret(ctx context.Context, cli k8sclient.Client, loggingScopeSecretName string) bool {
	secretKey := clusters.GetManagedClusterElasticsearchSecretKey()
	if loggingScopeSecretName != secretKey.Name {
		// We are not using the managed cluster Elasticsearch secret in our logging scope
		return false
	}
	secret := &corev1.Secret{}
	err := cli.Get(ctx, secretKey, secret)
	if kerrs.IsNotFound(err) {
		// Not a managed cluster - can't use managed cluster ES secret
		return false
	}
	// we retrieved the secret and there were no errors - we should use it
	return err == nil
}
