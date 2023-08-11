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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetVerrazzanoV1Alpha1(t *testing.T) {
	ctx := context.TODO()

	scheme := runtime.NewScheme()
	err := v1beta1.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// create the VZ resource. Stored as v1beta1.
	vzStoredV1Beta1, err := loadV1Beta1(testCaseBasic)
	assert.NoError(t, err)
	err = client.Create(ctx, vzStoredV1Beta1)
	assert.NoError(t, err)

	// the expected VZ resource returned should be v1alpha1
	vzExpected, err := loadV1Alpha1CR(testCaseBasic)
	assert.NoError(t, err)
	name := types.NamespacedName{
		Name:      vzExpected.Name,
		Namespace: vzExpected.Namespace,
	}

	// get the v1alpha1 VZ resource
	vzActual, err := GetVerrazzanoV1Alpha1(ctx, client, name)
	assert.NoError(t, err)

	// expected and actual v1alpha1 CRs must be equal
	assert.EqualValues(t, vzExpected.ObjectMeta.Name, vzActual.ObjectMeta.Name)
	assert.EqualValues(t, vzExpected.ObjectMeta.Namespace, vzActual.ObjectMeta.Namespace)
	assert.EqualValues(t, vzExpected.Spec, vzActual.Spec)
	assert.EqualValues(t, vzExpected.Status, vzActual.Status)
}

func TestGetVerrazzanoV1Alpha1NotFound(t *testing.T) {
	ctx := context.TODO()

	scheme := runtime.NewScheme()
	err := v1beta1.AddToScheme(scheme)
	assert.NoError(t, err)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// get the v1alpha1 VZ resource
	vzActual, err := GetVerrazzanoV1Alpha1(ctx, client, types.NamespacedName{
		Name:      "verrazzano",
		Namespace: "default",
	})

	// a NotFound error should have been returned
	assert.True(t, apierrors.IsNotFound(err), "a NotFound error was expected, but got '%v'", err)
	assert.Nil(t, vzActual)
}
