// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testLogPath     = "/foo/bar"
	testParseRules  = "test-parse-rules"
	testStorageName = "test-fluentd-volume"
	testESURL       = "http://es-host:9999"
	testESSecret    = "test-secret"

	testESURLUpdate    = "http://es-host-update:1111"
	testESSecretUpdate = "test-secret-update"

	testWorkLoadType = "test-workload"
)

// TestFluentdApply tests the creation of all FLUENTD resources in the system for a resource
// GIVEN a desired state for FLUENTD resources where no resources yet exist
// WHEN Apply is called
// THEN ensure that the expected FLUENTD resources are created
func TestFluentdApply(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	scope := createTestLoggingScope(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	fluentd := Fluentd{mockClient, ctrl.Log, context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	// simulate config map not existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, inNamespace client.InNamespace, fields client.MatchingFields) error {
			asserts.Equal(t, client.InNamespace(testNamespace), inNamespace)
			asserts.Equal(t, client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}, fields)
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())
			return nil
		})

	mockClient.EXPECT().
		Get(gomock.Any(), clusters.MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *v1.Secret) error {
			vmiSecret(sec)
			return nil
		})

	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configMap *v1.ConfigMap, options *client.CreateOptions) error {
			asserts.Equal(t, *fluentd.createFluentdConfigMap(testNamespace), *configMap)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// simulate Elasticsearch secret not existing
	testESSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: scope.Spec.SecretName}
	mockClient.EXPECT().
		Get(fluentd.Context, testESSecretFullName, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.ParseGroupResource("v1.Secret"), scope.Spec.SecretName))

	// expect empty Elasticsearch secret created in app namespace
	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, scope.Spec.SecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// invoke method being tested
	changesMade, err := fluentd.Apply(scope, resource, fluentdPod)

	asserts.True(t, changesMade)
	asserts.Nil(t, err)

	testAssertFluentdPodForApply(t, fluentdPod, testESSecret)

	mocker.Finish()
}

// TestFluentdApplyForUpdate tests the update of FLUENTD resources in the system for a resource
// GIVEN a desired state which contains updates for existing FLUENTD resources
// WHEN Apply is called
// THEN ensure that the expected FLUENTD resources updates occur
func TestFluentdApplyForUpdate(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	scope := createTestLoggingScope(true)
	updateLoggingScope(scope)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPodForUpdate()

	fluentd := Fluentd{mockClient, ctrl.Log, context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	mockClient.EXPECT().
		Get(gomock.Any(), clusters.MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *v1.Secret) error {
			vmiSecret(sec)
			return nil
		})

	// simulate config map existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, inNamespace client.InNamespace, fields client.MatchingFields) error {
			asserts.Equal(t, client.InNamespace(testNamespace), inNamespace)
			asserts.Equal(t, client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}, fields)
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())

			// this represents the found configmap resource. Only the length is checked not the item details
			resources.Items = append(resources.Items, unstructured.Unstructured{})

			return nil
		})

	// simulate Elasticsearch secret existing
	testESSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: scope.Spec.SecretName}
	mockClient.EXPECT().
		Get(fluentd.Context, testESSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = scope.Spec.SecretName
			secret.Namespace = testNamespace
			secret.Data = map[string][]byte{constants.ElasticsearchUsernameData: []byte("someuser")}
			return nil
		})

	// invoke method being tested
	changesMade, err := fluentd.Apply(scope, resource, fluentdPod)

	asserts.True(t, changesMade)
	asserts.Nil(t, err)

	testAssertFluentdPodForApplyUpdate(t, fluentdPod)

	mocker.Finish()
}

// TestFluentdApply tests the deletion of all FLUENTD resources in the system for a resource
// GIVEN a resource with associated FLUENTD resources which is no longer associated with a logging scope
// WHEN Remove is called
// THEN ensure that the expexted FLUENTD resources are removed
func TestFluentdRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	fluentd := &Fluentd{mockClient, ctrl.Log, context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}
	scope := createTestLoggingScope(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	mockClient.EXPECT().
		Get(gomock.Any(), clusters.MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *v1.Secret) error {
			vmiSecret(sec)
			return nil
		})

	addFluentdArtifactsToFluentdPod(fluentd, fluentdPod, scope, resource.Namespace)

	// simulate config map existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, inNamespace client.InNamespace, fields client.MatchingFields) error {
			asserts.Equal(t, client.InNamespace(testNamespace), inNamespace)
			asserts.Equal(t, client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}, fields)
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())

			// this represents the found configmap resource. Only the length is checked not the item details
			resources.Items = append(resources.Items, unstructured.Unstructured{})

			return nil
		})
	mockClient.EXPECT().
		Delete(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configmap *v1.ConfigMap, options *client.DeleteOptions) error {
			asserts.True(t, reflect.DeepEqual(fluentd.createFluentdConfigMap(testNamespace), configmap))
			asserts.Equal(t, client.DeleteOptions{}, *options)

			return nil
		})

	removeVerified := fluentd.Remove(scope, resource, fluentdPod)

	asserts.False(t, removeVerified)
	testAssertFluentdPodForRemove(t, fluentdPod)
	mocker.Finish()
}

// TestFluentdApply_ManagedClusterElasticsearch tests the creation of all FLUENTD resources in the \
// system for a resource on a MANAGED cluster
// GIVEN a desired state for FLUENTD resources where no resources yet exist
// WHEN Apply is called on a Managed Cluster using the default logging scope
// THEN ensure that the expected FLUENTD resources are created and the managed cluster's elasticsearch
// secret is used to determine the ES endpoint
func TestFluentdApply_ManagedClusterElasticsearch(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	managedClusterElasticsearchSecretKey := clusters.GetManagedClusterElasticsearchSecretKey()
	scope := createTestLoggingScope(true)
	// loggingscope uses managed cluster elasticsearch secret
	scope.Spec.SecretName = managedClusterElasticsearchSecretKey.Name
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	fluentd := Fluentd{mockClient, ctrl.Log, context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	// simulate config map not existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, inNamespace client.InNamespace, fields client.MatchingFields) error {
			asserts.Equal(t, client.InNamespace(testNamespace), inNamespace)
			asserts.Equal(t, client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}, fields)
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())
			return nil
		})

	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configMap *v1.ConfigMap, options *client.CreateOptions) error {
			asserts.Equal(t, *fluentd.createFluentdConfigMap(testNamespace), *configMap)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// Get cluster secret for cluster name
	mockClient.EXPECT().
		Get(gomock.Any(), clusters.MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, sec *v1.Secret) error {
			vmiSecret(sec)
			return nil
		})

	// simulate Elasticsearch secret not existing in app NS
	testESSecretFullName := types.NamespacedName{Namespace: testNamespace, Name: scope.Spec.SecretName}
	mockClient.EXPECT().
		Get(fluentd.Context, testESSecretFullName, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.ParseGroupResource("v1.Secret"), scope.Spec.SecretName))

	// simulate managed cluster ES secret existing (GET is called once to check if we should use
	// managed cluster, and once to actually perform the copy over to app NS)
	expectedData := map[string][]byte{constants.ElasticsearchUsernameData: []byte("someuser")}
	mockClient.EXPECT().
		Get(fluentd.Context, managedClusterElasticsearchSecretKey, gomock.Not(gomock.Nil())).
		Times(2).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = managedClusterElasticsearchSecretKey.Name
			secret.Namespace = managedClusterElasticsearchSecretKey.Namespace
			secret.Data = expectedData
			return nil
		})

	// expect copy of managed cluster ES secret created in app namespace
	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *v1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, testNamespace, sec.Namespace)
			asserts.Equal(t, managedClusterElasticsearchSecretKey.Name, sec.Name)
			asserts.Equal(t, expectedData, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})

	// invoke method being tested
	changesMade, err := fluentd.Apply(scope, resource, fluentdPod)

	asserts.True(t, changesMade)
	asserts.Nil(t, err)

	testAssertFluentdPodForApply(t, fluentdPod, managedClusterElasticsearchSecretKey.Name)

	mocker.Finish()
}

// createTestFluentdPod creates a test FluentdPod
func createTestFluentdPod() *FluentdPod {
	return &FluentdPod{
		Containers:   []v1.Container{{Name: "test-container"}},
		Volumes:      []v1.Volume{{Name: "test-volume"}},
		VolumeMounts: []v1.VolumeMount{{Name: "test-volume-mount"}},
		HandlerEnv:   []v1.EnvVar{{Name: "test-env-var", Value: "test-env-value"}},
		LogPath:      testLogPath,
	}
}

// createTestFluendPodForUpdate creates a test FluentdPod to be used in testing update
func createTestFluentdPodForUpdate() *FluentdPod {
	env := createFluentdESEnv()
	fluentdContainer := v1.Container{Name: fluentdContainerName, Env: env}
	fluentdPod := &FluentdPod{
		Containers:   []v1.Container{{Name: "test-container"}, fluentdContainer},
		Volumes:      []v1.Volume{{Name: "test-volume"}},
		VolumeMounts: []v1.VolumeMount{{Name: "test-volume-mount"}},
		HandlerEnv:   []v1.EnvVar{{Name: "test-env-var", Value: "test-env-value"}},
		LogPath:      testLogPath,
	}
	return fluentdPod
}

// addFluentdArtifactsToFluentdPod adds FLUENTD artifacts to a FluentdPod
func addFluentdArtifactsToFluentdPod(fluentd *Fluentd, fluentdPod *FluentdPod, scope *v1alpha1.LoggingScope, namespace string) {
	fluentd.ensureFluentdVolumes(fluentdPod, scope)
	fluentdPod.VolumeMounts = append(fluentdPod.VolumeMounts, fluentd.createStorageVolumeMount())
	fluentdPod.Containers = append(fluentdPod.Containers, fluentd.createFluentdContainer(fluentdPod, scope, namespace))
}

// testAssertFluentdPodForApply asserts FluentdPod state for Apply
func testAssertFluentdPodForApply(t *testing.T, fluentdPod *FluentdPod, esSecretName string) {
	containers := fluentdPod.Containers
	asserts.Len(t, containers, 2)
	success := false
	for _, container := range containers {
		if container.Name == fluentdContainerName {
			env := container.Env
			for _, envVar := range env {
				switch name := envVar.Name; name {
				case elasticSearchURLEnv:
					asserts.Equal(t, testESURL, envVar.Value)
				case elasticSearchUserEnv:
					asserts.Equal(t, esSecretName, envVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				case elasticSearchPwdEnv:
					asserts.Equal(t, esSecretName, envVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				}

			}
			success = true
		}
	}
	asserts.True(t, success)

	volumes := fluentdPod.Volumes
	asserts.Len(t, volumes, 4)

	volumeMounts := fluentdPod.VolumeMounts
	asserts.Len(t, volumeMounts, 2)
}

// testAssertFluentdPodForApply asserts FluentdPod state for Apply updates
func testAssertFluentdPodForApplyUpdate(t *testing.T, fluentdPod *FluentdPod) {
	containers := fluentdPod.Containers
	asserts.Len(t, containers, 2)
	success := false
	for _, container := range containers {
		if container.Name == fluentdContainerName {
			env := container.Env
			for _, envVar := range env {
				switch name := envVar.Name; name {
				case elasticSearchURLEnv:
					asserts.Equal(t, testESURLUpdate, envVar.Value)
				case elasticSearchUserEnv:
					asserts.Equal(t, testESSecretUpdate, envVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				case elasticSearchPwdEnv:
					asserts.Equal(t, testESSecretUpdate, envVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
				}

			}
			success = true
		}
	}
	asserts.True(t, success)

	volumes := fluentdPod.Volumes
	asserts.Len(t, volumes, 4)

	volumeMounts := fluentdPod.VolumeMounts
	asserts.Len(t, volumeMounts, 2)
}

// testAssertFluentdPodForRemove asserts FluendPod state for Remove
func testAssertFluentdPodForRemove(t *testing.T, fluentdPod *FluentdPod) {
	asserts.Len(t, fluentdPod.Containers, 1)
	// WebLogic FLUENTD volumes don't get removed as a result of disassociation from scope
	asserts.Len(t, fluentdPod.Volumes, 3)
	asserts.Len(t, fluentdPod.VolumeMounts, 2)
}

// createFluentdESEnv creates Env Var set
func createFluentdESEnv() []v1.EnvVar {
	return []v1.EnvVar{
		{
			Name:  "LOG_PATH",
			Value: testLogPath,
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
			Value: testESURL,
		},
		{
			Name: elasticSearchUserEnv,
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: testESSecret,
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
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: testESSecret,
					},
					Key: constants.ElasticsearchPasswordData,
					Optional: func(opt bool) *bool {
						return &opt
					}(true),
				},
			},
		},
	}
}
