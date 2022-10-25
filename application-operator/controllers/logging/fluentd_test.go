// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logging

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testLogPath      = "/foo/bar"
	testParseRules   = "test-parse-rules"
	testStorageName  = "test-fluentd-volume"
	testWorkLoadType = "test-workload"
)

// TestFluentdApply tests the creation of all FLUENTD resources in the system for a resource
// GIVEN a desired state for FLUENTD resources where no resources yet exist
// WHEN Apply is called
// THEN ensure that the expected FLUENTD resources are created
func TestFluentdApply(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	logInfo := createTestLogInfo(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	fluentd := Fluentd{mockClient, zap.S(), context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	// simulate config map not existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, opts ...client.ListOption) error {
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())
			return nil
		})

	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configMap *v1.ConfigMap, options ...client.CreateOption) error {
			asserts.Equal(t, *fluentd.createFluentdConfigMap(testNamespace), *configMap)
			return nil
		})

	// invoke method being tested
	err := fluentd.Apply(logInfo, resource, fluentdPod)

	asserts.Nil(t, err)
	testAssertFluentdPodForApply(t, fluentdPod)

	mocker.Finish()
}

// TestFluentdApplyForUpdate tests the update of FLUENTD resources in the system for a resource
// GIVEN a desired state which contains updates for existing FLUENTD resources
// WHEN Apply is called
// THEN ensure that the expected FLUENTD resources updates occur
func TestFluentdApplyForUpdate(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	logInfo := createTestLogInfo(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPodForUpdate()

	fluentd := Fluentd{mockClient, zap.S(), context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	// simulate config map existing

	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, opts ...client.ListOption) error {
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())

			// this represents the found configmap resource. Only the length is checked not the item details
			resources.Items = append(resources.Items, unstructured.Unstructured{})

			return nil
		})
	mockClient.EXPECT().
		Update(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configMap *v1.ConfigMap, options ...client.UpdateOption) error {
			asserts.Equal(t, *fluentd.createFluentdConfigMap(testNamespace), *configMap)
			return nil
		})

	// invoke method being tested
	err := fluentd.Apply(logInfo, resource, fluentdPod)

	asserts.Nil(t, err)
	testAssertFluentdPodForApplyUpdate(t, fluentdPod)

	mocker.Finish()
}

// TestFluentdApply tests the deletion of all FLUENTD resources in the system for a resource
// GIVEN a resource with associated FLUENTD resources which is no longer associated with a logging logInfo
// WHEN Remove is called
// THEN ensure that the expexted FLUENTD resources are removed
func TestFluentdRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	fluentd := &Fluentd{mockClient, zap.S(), context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}
	logInfo := createTestLogInfo(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	addFluentdArtifactsToFluentdPod(fluentd, fluentdPod, logInfo, resource.Namespace)

	// simulate config map existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, opts ...client.ListOption) error {
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())

			// this represents the found configmap resource. Only the length is checked not the item details
			resources.Items = append(resources.Items, unstructured.Unstructured{})

			return nil
		})
	mockClient.EXPECT().
		Delete(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configmap *v1.ConfigMap, options ...client.DeleteOption) error {
			asserts.True(t, reflect.DeepEqual(fluentd.createFluentdConfigMap(testNamespace), configmap))
			return nil
		})

	removeVerified := fluentd.Remove(logInfo, resource, fluentdPod)

	asserts.False(t, removeVerified)
	testAssertFluentdPodForRemove(t, fluentdPod)
	mocker.Finish()
}

// TestFluentdApply_ManagedClusterElasticsearch tests the creation of all FLUENTD resources in the \
// system for a resource on a MANAGED cluster
// GIVEN a desired state for FLUENTD resources where no resources yet exist
// WHEN Apply is called on a Managed Cluster using the default logging logInfo
// THEN ensure that the expected FLUENTD resources are created and the managed cluster's elasticsearch
// secret is used to determine the ES endpoint
func TestFluentdApply_ManagedClusterElasticsearch(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	logInfo := createTestLogInfo(true)
	resource := createTestResourceRelation()
	fluentdPod := createTestFluentdPod()

	fluentd := Fluentd{mockClient, zap.S(), context.Background(), testParseRules, testStorageName, scratchVolMountPath, testWorkLoadType}

	// simulate config map not existing
	mockClient.EXPECT().
		List(fluentd.Context, gomock.Not(gomock.Nil()), client.InNamespace(testNamespace), client.MatchingFields{"metadata.name": configMapName + "-" + testWorkLoadType}).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, opts ...client.ListOption) error {
			asserts.Equal(t, configMapAPIVersion, resources.GetAPIVersion())
			asserts.Equal(t, configMapKind, resources.GetKind())
			return nil
		})

	mockClient.EXPECT().
		Create(fluentd.Context, gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, configMap *v1.ConfigMap, options ...client.CreateOption) error {
			asserts.Equal(t, *fluentd.createFluentdConfigMap(testNamespace), *configMap)
			return nil
		})

	// invoke method being tested
	err := fluentd.Apply(logInfo, resource, fluentdPod)

	asserts.Nil(t, err)
	testAssertFluentdPodForApply(t, fluentdPod)

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
	fluentdContainer := v1.Container{Name: FluentdStdoutSidecarName, Env: env}
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
func addFluentdArtifactsToFluentdPod(fluentd *Fluentd, fluentdPod *FluentdPod, logInfo *LogInfo, namespace string) {
	fluentd.ensureFluentdVolumes(fluentdPod)
	fluentdPod.VolumeMounts = append(fluentdPod.VolumeMounts, fluentd.createStorageVolumeMount())
	fluentdPod.Containers = append(fluentdPod.Containers, fluentd.createFluentdContainer(fluentdPod, logInfo, namespace))
}

// testAssertFluentdPodForApply asserts FluentdPod state for Apply
func testAssertFluentdPodForApply(t *testing.T, fluentdPod *FluentdPod) {
	containers := fluentdPod.Containers
	asserts.Len(t, containers, 2)

	volumes := fluentdPod.Volumes
	asserts.Len(t, volumes, 3)

	volumeMounts := fluentdPod.VolumeMounts
	asserts.Len(t, volumeMounts, 2)
}

// testAssertFluentdPodForApply asserts FluentdPod state for Apply updates
func testAssertFluentdPodForApplyUpdate(t *testing.T, fluentdPod *FluentdPod) {
	containers := fluentdPod.Containers
	asserts.Len(t, containers, 2)

	volumes := fluentdPod.Volumes
	asserts.Len(t, volumes, 3)

	volumeMounts := fluentdPod.VolumeMounts
	asserts.Len(t, volumeMounts, 2)
}

// testAssertFluentdPodForRemove asserts FluendPod state for Remove
func testAssertFluentdPodForRemove(t *testing.T, fluentdPod *FluentdPod) {
	asserts.Len(t, fluentdPod.Containers, 1)
	// WebLogic FLUENTD volumes don't get removed as a result of disassociation from logInfo
	asserts.Len(t, fluentdPod.Volumes, 2)
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
	}
}

// createTestResourceRelation creates a new test QualifiedResourceRelation
func createTestResourceRelation() vzapi.QualifiedResourceRelation {
	resource := vzapi.QualifiedResourceRelation{
		APIVersion: testAPIVersion,
		Kind:       "Domain",
		Namespace:  testNamespace,
		Name:       "testName",
		Role:       "",
	}

	return resource
}
