// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	"sigs.k8s.io/yaml"
)

// TestCreateEnvoyFilter tests creating the Envoy filter
// GIVEN a client
//  WHEN I call createEnvoyFilter when the filter doesn't exist
//  THEN the bash function is called to create the filter
func TestCreateEnvoyFilter(t *testing.T) {
	a := assert.New(t)
	a.Equal("istio", comp.Name(), "Wrong component name")

	ctx := spi.NewFakeContext(getIstioFilterMockCreate(t), nil, nil, false)
	err := createEnvoyFilter(ctx.Log(), ctx.Client())
	a.NoError(err, "Error %s calling createEnvoyFilter", err)
}

func getIstioFilterMockCreate(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// expect a call to get the filter
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			return k8serr.NewNotFound(schema.GroupResource{Group: "networking.istio.io/v1alpha3", Resource: "EnvoyFilter"}, name.Name)
		})

	// expect a call to create the filter
	mock.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			return validateSpec(ctx, u)
		})
	return mock
}

// TestUpdateEnvoyFilter tests updating the Envoy filter
// GIVEN a client
//  WHEN I call createEnvoyFilter when the filter exists
//  THEN the bash function is called to update the filter
func TestUpdateEnvoyFilter(t *testing.T) {
	a := assert.New(t)
	a.Equal("istio", comp.Name(), "Wrong component name")

	ctx := spi.NewFakeContext(getIstioFilterMockUpdate(t), nil, nil, false)
	err := createEnvoyFilter(ctx.Log(), ctx.Client())
	a.NoError(err, "Error %s calling createEnvoyFilter", err)
}

func getIstioFilterMockUpdate(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// expect a call to get the filter
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			return nil
		})

	// expect a call to update the filter
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.UpdateOption) error {
			return validateSpec(ctx, u)
		})

	return mock
}

func validateSpec(ctx context.Context, u *unstructured.Unstructured) error {
	// Get the spec field
	inputSpec, _, err := unstructured.NestedFieldNoCopy(u.Object, specField)
	if err != nil {
		return err
	}
	// Get the expected Spec YAML
	expectedFilter := &unstructured.Unstructured{Object: map[string]interface{}{}}
	err = yaml.Unmarshal([]byte(filterYaml), expectedFilter)
	if err != nil {
		return err
	}
	expectedSpec, _, err := unstructured.NestedFieldNoCopy(u.Object, specField)
	if err != nil {
		return err
	}
	if !equality.Semantic.DeepEqual(expectedSpec, inputSpec) {
		return errors.New("Envoy filter spec has wrong value")
	}
	return nil
}
