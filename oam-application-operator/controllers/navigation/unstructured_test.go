// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"encoding/json"
	"fmt"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/oam-application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestGetKindOfUnstructured tests the GetKindOfUnstructured function.
func TestGetKindOfUnstructured(t *testing.T) {
	assert := asserts.New(t)

	var uns unstructured.Unstructured
	var kind string
	var err error

	// GIVEN an unstructured with a valid kind
	// WHEN the kind is extracted
	// THEN verify that the kind is correct and there is no error
	uns = unstructured.Unstructured{}
	uns.SetGroupVersionKind(k8sapps.SchemeGroupVersion.WithKind("Deployment"))
	kind, err = GetKindOfUnstructured(&uns)
	assert.NoError(err)
	assert.Equal("Deployment", kind)

	// GIVEN an unstructured without a valid kind
	// WHEN the kind is extracted
	// THEN verify that the kind is empty and that an error was returned
	uns = unstructured.Unstructured{}
	kind, err = GetKindOfUnstructured(&uns)
	assert.Error(err)
	assert.Contains(err.Error(), "kind")
	assert.Equal("", kind)

	// GIVEN a nil input unstructured parameter
	// WHEN the kind is extracted
	// THEN verify that an error is returned
	kind, err = GetKindOfUnstructured(nil)
	assert.Error(err)
	assert.Equal("", kind)
}

// TestGetAPIVersionOfUnstructured tests the GetAPIVersionOfUnstructured function.
func TestGetAPIVersionOfUnstructured(t *testing.T) {
	assert := asserts.New(t)

	var uns unstructured.Unstructured
	var apiver string
	var err error

	// GIVEN a nil unstructured input parameter
	// WHEN the APIVersion is extracted
	// THEN verify an error is returned
	apiver, err = GetAPIVersionOfUnstructured(nil)
	assert.Error(err)
	assert.Equal("", apiver)

	// GIVEN a nil unstructured without an api version
	// WHEN the APIVersion is extracted
	// THEN verify an error is returned
	uns = unstructured.Unstructured{}
	apiver, err = GetAPIVersionOfUnstructured(&uns)
	assert.Error(err)
	assert.Contains("unstructured does not contain api version", err.Error())
	assert.Equal("", apiver)

	// GIVEN a nil unstructured with an api version
	// WHEN the APIVersion is extracted
	// THEN verify the api version is correct and there is no error
	uns = unstructured.Unstructured{}
	uns.SetGroupVersionKind(k8sapps.SchemeGroupVersion.WithKind("Deployment"))
	apiver, err = GetAPIVersionOfUnstructured(&uns)
	assert.NoError(err)
	assert.Equal("apps/v1", apiver)
}

// TestGetAPIVersionKindOfUnstructured tests the GetAPIVersionKindOfUnstructured function
func TestGetAPIVersionKindOfUnstructured(t *testing.T) {
	assert := asserts.New(t)

	var uns unstructured.Unstructured
	var avk string
	var err error

	// GIVEN a nil unstructured parameter
	// WHEN the api version kind is extracted
	// THEN verify an error is returned
	avk, err = GetAPIVersionKindOfUnstructured(nil)
	assert.Error(err)
	assert.Equal("", avk)

	// GIVEN an invalid unstructured parameter
	// WHEN the api version kind is extracted
	// THEN verify an error is returned
	uns = unstructured.Unstructured{}
	avk, err = GetAPIVersionKindOfUnstructured(&uns)
	assert.Error(err)
	assert.Equal("", avk)

	// GIVEN an unstructured parameter with an invalid api version kind
	// WHEN the api version kind is extracted
	// THEN verify an error is returned
	uns = unstructured.Unstructured{}
	uns.SetAPIVersion(k8sapps.SchemeGroupVersion.String())
	avk, err = GetAPIVersionKindOfUnstructured(&uns)
	assert.Error(err)
	assert.Equal("", avk)

	// GIVEN an unstructured parameter with an valid api version kind
	// WHEN the api version kind is extracted
	// THEN verify the correct api version kind is returned
	uns = unstructured.Unstructured{}
	uns.SetGroupVersionKind(k8sapps.SchemeGroupVersion.WithKind("Deployment"))
	avk, err = GetAPIVersionKindOfUnstructured(&uns)
	assert.NoError(err)
	assert.Equal("apps/v1.Deployment", avk)
}

// TestGetUnstructuredChildResourcesByAPIVersionKindsPositive tests the FetchUnstructuredChildResourcesByAPIVersionKinds function.
// GIVEN a valid list of child resources
// WHEN a request is made to list those child resources
// THEN verify that the children are returned
func TestGetUnstructuredChildResourcesByAPIVersionKindsPositive(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var children []*unstructured.Unstructured

	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		List(gomock.Eq(ctx), gomock.Not(gomock.Nil()), gomock.Eq(client.InNamespace("test-namespace")), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, namespace client.InNamespace, labels client.MatchingLabels) error {
			assert.Equal("Deployment", resources.GetKind())
			return AppendAsUnstructured(resources, k8sapps.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: k8sapps.SchemeGroupVersion.String(),
					Kind:       "test-invalid-kind"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment-name",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: oamcore.ContainerizedWorkloadKindAPIVersion,
						Kind:       oamcore.ContainerizedWorkloadKind,
						Name:       "test-workload-name",
						UID:        "test-workload-uid"}}}})
		})
	children, err = FetchUnstructuredChildResourcesByAPIVersionKinds(ctx, cli, ctrl.Log, "test-namespace", "test-workload-uid", []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}})
	mocker.Finish()
	assert.NoError(err)
	assert.Len(children, 1)
}

// TestGetUnstructuredChildResourcesByAPIVersionKindsPositive tests the FetchUnstructuredChildResourcesByAPIVersionKinds function.
// GIVEN a request to list child resources
// WHEN a the underlying kubernetes call fails
// THEN verify that the error is propigated to the caller
func TestFetchUnstructuredChildResourcesByAPIVersionKindsNegative(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var children []*unstructured.Unstructured

	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		List(gomock.Eq(ctx), gomock.Not(gomock.Nil()), gomock.Eq(client.InNamespace("test-namespace")), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, namespace client.InNamespace, labels client.MatchingLabels) error {
			return fmt.Errorf("test-error")
		})
	children, err = FetchUnstructuredChildResourcesByAPIVersionKinds(ctx, cli, ctrl.Log, "test-namespace", "test-workload-uid", []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}})
	mocker.Finish()
	assert.Error(err)
	assert.Equal("test-error", err.Error())
	assert.Len(children, 0)
}

// TestFetchUnstructuredByReference tests the FetchUnstructuredByReference function
func TestFetchUnstructuredByReference(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var uns *unstructured.Unstructured

	// GIVEN a valid reference
	// WHEN an underlying k8s api call fails
	// THEN propagate the error to the caller
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-space", Name: "test-name"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, uns *unstructured.Unstructured) error {
			return fmt.Errorf("test-error")
		})
	uns, err = FetchUnstructuredByReference(ctx, cli, ctrl.Log, v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-api/ver",
		Kind:       "test-kind",
		Namespace:  "test-space",
		Name:       "test-name",
		Role:       "test-role"})
	mocker.Finish()
	assert.Nil(uns)
	assert.Error(err)

	// GIVEN a valid reference
	// WHEN an unstructured resource is requested for the reference
	// THEN verify the returned unstructured resource has correct information
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-space", Name: "test-name"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, uns *unstructured.Unstructured) error {
			uns.SetNamespace(key.Namespace)
			uns.SetName(key.Name)
			return nil
		})
	uns, err = FetchUnstructuredByReference(ctx, cli, ctrl.Log, v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-api/ver",
		Kind:       "test-kind",
		Namespace:  "test-space",
		Name:       "test-name",
		Role:       "test-role"})
	mocker.Finish()
	assert.NotNil(uns)
	assert.NoError(err)
	assert.Equal("test-space", uns.GetNamespace())
	assert.Equal("test-name", uns.GetName())
}

// ConvertToUnstructured converts an object to an Unstructured version
// object - The object to convert to Unstructured
func ConvertToUnstructured(object interface{}) (unstructured.Unstructured, error) {
	bytes, err := json.Marshal(object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	var u map[string]interface{}
	json.Unmarshal(bytes, &u)
	return unstructured.Unstructured{Object: u}, nil
}

// AppendAsUnstructured appends an object to the list after converting it to an Unstructured
// list - The list to append to.
// object - The object to convert to Unstructured and append to the list
func AppendAsUnstructured(list *unstructured.UnstructuredList, object interface{}) error {
	u, err := ConvertToUnstructured(object)
	if err != nil {
		return err
	}
	list.Items = append(list.Items, u)
	return nil
}
