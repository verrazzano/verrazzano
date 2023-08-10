// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetVerrazzanoV1Alpha1(t *testing.T) {
	// FIXME: make it cleaner how I create this
	vzExpected, err := loadV1Alpha1CR(testCaseBasic)
	assert.NoError(t, err)

	ctx := context.TODO()

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	assert.NoError(t, err)
	err = clientgoscheme.AddToScheme(scheme)
	assert.NoError(t, err)
	err = v1beta1.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// create the VZ resource
	err = client.Create(ctx, vzExpected)
	assert.NoError(t, err)

	// get the v1alpha1 VZ resource
	vzActual, err := GetVerrazzanoV1Alpha1(ctx, client, types.NamespacedName {
		Name: vzExpected.Name,
		Namespace: vzExpected.Namespace,
	})
	assert.NoError(t, err)

	// expected and actual v1alpha1 CRs must be equal
	assert.EqualValues(t, vzExpected.TypeMeta, vzActual.TypeMeta)
	assert.EqualValues(t, vzExpected.ObjectMeta, vzActual.ObjectMeta)
	assert.EqualValues(t, vzExpected.Spec, vzActual.Spec)
	assert.EqualValues(t, vzExpected.Status, vzActual.Status)
}

func TestGetVerrazzanoV1Alpha1NotFound(t *testing.T) {
	ctx := context.TODO()

	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	assert.NoError(t, err)
	err = clientgoscheme.AddToScheme(scheme)
	assert.NoError(t, err)
	err = v1beta1.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// get the v1alpha1 VZ resource
	vzActual, err := GetVerrazzanoV1Alpha1(ctx, client, types.NamespacedName {
		Name: "verrazzano",
		Namespace: "default",
	})
	
	// a NotFound error should have been returned
	assert.True(t, apierrors.IsNotFound(err))
	assert.Nil(t, vzActual)
}