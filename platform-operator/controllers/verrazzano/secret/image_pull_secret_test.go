// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package secret

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCopyPullSecret tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN the secret does not exist in the target namespace
// THEN true is returned
func TestCopyPullSecret(t *testing.T) {
	name := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace},
		},
	).Build()
	copied, err := CheckImagePullSecret(fakeClient, constants.VerrazzanoSystemNamespace)
	assert.NoError(t, err)
	assert.True(t, copied)
}

// TestCopyGlobalPullSecretDoesNotExist tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN the source secret does not exist in the default namespace
// THEN false is returned
func TestCopyGlobalPullSecretDoesNotExist(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	copied, err := CheckImagePullSecret(fakeClient, constants.VerrazzanoSystemNamespace)
	assert.NoError(t, err)
	assert.False(t, copied)
}

// TestTargetPullSecretAlreadyExists tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN the pull secret already exists in the target namespace
// THEN true is returned
func TestTargetPullSecretAlreadyExists(t *testing.T) {
	name := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: constants.VerrazzanoSystemNamespace}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: "default"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace}},
	).Build()
	copied, err := CheckImagePullSecret(fakeClient, constants.VerrazzanoSystemNamespace)
	assert.NoError(t, err)
	assert.True(t, copied)
}

// TestUnexpectedErrorGetTargetSecret tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN an unexpected error is returned on the global secret check
// THEN false and an error are returned
func TestUnexpectedErrorGetTargetSecret(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	name := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}

	// Expect a call to get an existing configmap, but return a NotFound error.
	mock.EXPECT().
		Get(gomock.Any(), name, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, _ client.ObjectKey, _ *corev1.Secret) error {
			return fmt.Errorf("Unexpected error")
		})
	copied, err := CheckImagePullSecret(mock, constants.VerrazzanoSystemNamespace)
	assert.NotNil(t, err)
	assert.False(t, copied)
}

// TestUnexpectedErrorOnCreate tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN an unexpected error is returned on the create target secret operation
// THEN false and an error are returned
func TestUnexpectedErrorOnCreate(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	targetSecretName := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: constants.VerrazzanoSystemNamespace}
	defaultName := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}

	// Expect a call to get the target ns secret first, return not found
	mock.EXPECT().
		Get(gomock.Any(), targetSecretName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			return errors.NewNotFound(schema.GroupResource{Group: constants.VerrazzanoSystemNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName)
		})
	// Expect a call to get the default ns secret.
	mock.EXPECT().
		Get(gomock.Any(), defaultName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			secret.ObjectMeta.Namespace = defaultName.Namespace
			secret.ObjectMeta.Name = defaultName.Name
			secret.Type = "kubernetes.io/dockerconfigjson"
			return nil
		})
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			return fmt.Errorf("Unexpected error")
		})
	copied, err := CheckImagePullSecret(mock, constants.VerrazzanoSystemNamespace)
	assert.NotNil(t, err)
	assert.False(t, copied)
}

// TestUnexpectedErrorGetSourceSecret tests a deployment ready status check
// GIVEN a call to checkImagePullSecret
// WHEN an unexpected error is returned on the get source secret operation
// THEN false and an error are returned
func TestUnexpectedErrorGetSourceSecret(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defaultName := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}

	// Expect a call to get the default ns secret.
	mock.EXPECT().
		Get(gomock.Any(), defaultName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			return fmt.Errorf("unexpected error")
		})
	copied, err := CheckImagePullSecret(mock, constants.VerrazzanoSystemNamespace)
	assert.NotNil(t, err)
	assert.False(t, copied)
}

// TestAddImagePullSecretUnexpectedError tests a deployment ready status check
// GIVEN a call to AddGlobalImagePullSecretHelmOverride
// WHEN an unexpected error is returned from checkImagePullSecret
// THEN an error is returned and the Key/Value pairs are unmodified
func TestAddImagePullSecretUnexpectedError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	targetSecretName := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}

	// Expect a call to get the target ns secret first, return not found
	mock.EXPECT().
		Get(gomock.Any(), targetSecretName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			return fmt.Errorf("unexpected error")
		}).AnyTimes()

	kvs := []bom.KeyValue{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}
	retKVs, err := AddGlobalImagePullSecretHelmOverride(vzlog.DefaultLogger(), mock, constants.VerrazzanoSystemNamespace, kvs, "helmKey")
	assert.Error(t, err)
	assert.Equal(t, kvs, retKVs)
}

// TestAddImagePullSecretTargetSecretAlreadyExists tests a deployment ready status check
// GIVEN a call to AddGlobalImagePullSecretHelmOverride
// WHEN the target secret already exists
// THEN no error is returned and the target secret helm Key/Value pair have been added to the Key/Value list
func TestAddImagePullSecretTargetSecretAlreadyExists(t *testing.T) {

	name := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: constants.VerrazzanoSystemNamespace}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.GlobalImagePullSecName, Namespace: "default"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace}},
	).Build()
	kvs := []bom.KeyValue{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}
	retKVs, err := AddGlobalImagePullSecretHelmOverride(vzlog.DefaultLogger(), fakeClient, constants.VerrazzanoSystemNamespace, kvs, "helmKey")
	assert.Nil(t, err)
	assert.Lenf(t, retKVs, 3, "Unexpected number of Key/Value pairs: %s", len(retKVs))
	for _, kv := range retKVs {
		assert.Containsf(t, []string{"key1", "key2", "helmKey"}, kv.Key, "Did not have Key %s", kv.Key)
		assert.Containsf(t, []string{"value1", "value2", constants.GlobalImagePullSecName}, kv.Value, "Did not have Value", kv.Value)
	}
}

// TestAddImagePullSecretTargetSecretCopied tests a deployment ready status check
// GIVEN a call to AddGlobalImagePullSecretHelmOverride
// WHEN the target secret is successfully copied
// THEN no error is returned and the target secret helm Key/Value pair have been added to the Key/Value list
func TestAddImagePullSecretTargetSecretCopied(t *testing.T) {

	name := types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace},
		},
	).Build()
	kvs := []bom.KeyValue{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}
	retKVs, err := AddGlobalImagePullSecretHelmOverride(vzlog.DefaultLogger(), fakeClient, constants.VerrazzanoSystemNamespace, kvs, "helmKey")
	assert.Nil(t, err)
	assert.Lenf(t, retKVs, 3, "Unexpected number of Key/Value pairs: %s", len(retKVs))
	for _, kv := range retKVs {
		assert.Containsf(t, []string{"key1", "key2", "helmKey"}, kv.Key, "Did not have Key %s", kv.Key)
		assert.Containsf(t, []string{"value1", "value2", constants.GlobalImagePullSecName}, kv.Value, "Did not have Value", kv.Value)
	}
}
