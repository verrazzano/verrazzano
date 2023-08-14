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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	nonexistentVZName = "nonexistent-verrazzano"
)

func TestGetVerrazzanoV1Alpha1(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// create a v1beta1 Verrazzano through the K8s client
	vzExpected, err := createTestVZ(ctx, client)
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
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// get the v1alpha1 VZ resource, which was never created
	vzActual, err := GetVerrazzanoV1Alpha1(ctx, client, types.NamespacedName{
		Name:      nonexistentVZName,
		Namespace: "",
	})

	// a NotFound error should have been returned
	assert.True(t, apierrors.IsNotFound(err), "a NotFound error was expected, but got '%v'", err)
	assert.Nil(t, vzActual)
}

func TestListVerrazzanoV1Alpha1(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// create a v1beta1 Verrazzano through the K8s client
	vzExpected, err := createTestVZ(ctx, client)
	assert.NoError(t, err)

	// get the v1alpha1 VZ resource
	vzList, err := ListVerrazzanoV1Alpha1(ctx, client)
	assert.NoError(t, err)
	expectedLength := 1
	assert.Len(t, vzList.Items, expectedLength, "the VerrazzanoList should have a length of %d but was %d", expectedLength, len(vzList.Items))

	// expected and actual v1alpha1 CRs must be equal
	vzActual := vzList.Items[0]
	assert.EqualValues(t, vzExpected.ObjectMeta.Name, vzActual.ObjectMeta.Name)
	assert.EqualValues(t, vzExpected.ObjectMeta.Namespace, vzActual.ObjectMeta.Namespace)
	assert.EqualValues(t, vzExpected.Spec, vzActual.Spec)
	assert.EqualValues(t, vzExpected.Status, vzActual.Status)
}

func TestListVerrazzanoV1Alpha1NotFound(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// list the VZ resources
	vzList, err := ListVerrazzanoV1Alpha1(ctx, client)
	assert.NoError(t, err)
	expectedLength := 0
	assert.Len(t, vzList.Items, expectedLength, "the VerrazzanoList should have a length of %d but was %d", expectedLength, len(vzList.Items))
}

func TestUpdateVerrazzanoV1Alpha1(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// create the VZ resource. Stored as v1beta1.
	vzV1Alpha1, err := createTestVZ(ctx, client)
	assert.NoError(t, err)

	// Get the VZ resource as it is before the Update
	vzNamespacedName := types.NamespacedName{
		Name:      vzV1Alpha1.Name,
		Namespace: vzV1Alpha1.Namespace,
	}
	vzV1Alpha1, err = GetVerrazzanoV1Alpha1(ctx, client, vzNamespacedName)
	assert.NoError(t, err)

	// Update the Verrazzano struct - add a new label
	labels := map[string]string{"dummy-label-key": "dummy-label-value"}
	vzV1Alpha1.SetLabels(labels)

	// Update the Verrazzano resource through the K8s client
	err = UpdateVerrazzanoV1Alpha1(ctx, client, vzV1Alpha1)
	assert.NoError(t, err)

	// Get the Verrazzano after the update
	vzRetrieved, err := GetVerrazzanoV1Alpha1(ctx, client, vzNamespacedName)
	assert.NoError(t, err)

	// The retrieved Verrazzano should have the updated label
	assert.EqualValues(t, vzV1Alpha1.ObjectMeta.Labels, vzRetrieved.ObjectMeta.Labels)

	// Check that other things from the retrieved Verrazzano are as expected
	assert.EqualValues(t, vzV1Alpha1.ObjectMeta.Name, vzRetrieved.ObjectMeta.Name)
	assert.EqualValues(t, vzV1Alpha1.ObjectMeta.Namespace, vzRetrieved.ObjectMeta.Namespace)
	assert.EqualValues(t, vzV1Alpha1.Spec, vzRetrieved.Spec)
	assert.EqualValues(t, vzV1Alpha1.Status, vzRetrieved.Status)
}

func TestUpdateVerrazzanoV1Alpha1NotFound(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	vzV1Alpha1, err := loadV1Alpha1CR(testCaseBasic)
	assert.NoError(t, err)

	// Attempt to update a nonexistent Verrazzano resource through the K8s client
	err = UpdateVerrazzanoV1Alpha1(ctx, client, vzV1Alpha1)

	// a NotFound error should have been returned
	assert.True(t, apierrors.IsNotFound(err), "a NotFound error was expected, but got '%v'", err)
}

func TestUpdateStatusVerrazzanoV1Alpha1(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	// create the VZ resource. Stored as v1beta1.
	vzV1Alpha1, err := createTestVZ(ctx, client)
	assert.NoError(t, err)

	// Get the VZ resource as it is before the Update
	vzNamespacedName := types.NamespacedName{
		Name:      vzV1Alpha1.Name,
		Namespace: vzV1Alpha1.Namespace,
	}
	vzV1Alpha1, err = GetVerrazzanoV1Alpha1(ctx, client, vzNamespacedName)
	assert.NoError(t, err)

	// Update the Verrazzano struct - change VZ State
	vzV1Alpha1.Status.State = VzStatePaused

	// Update the Verrazzano resource through the K8s client
	err = UpdateVerrazzanoV1Alpha1Status(ctx, client.Status(), vzV1Alpha1)
	assert.NoError(t, err)

	// Get the Verrazzano after the update
	vzRetrieved, err := GetVerrazzanoV1Alpha1(ctx, client, vzNamespacedName)
	assert.NoError(t, err)

	// The retrieved Verrazzano should have the updated status state
	assert.EqualValues(t, vzV1Alpha1.Status.State, vzRetrieved.Status.State)

	// Check that other things from the retrieved Verrazzano are as expected
	assert.EqualValues(t, vzV1Alpha1.ObjectMeta.Name, vzRetrieved.ObjectMeta.Name)
	assert.EqualValues(t, vzV1Alpha1.ObjectMeta.Namespace, vzRetrieved.ObjectMeta.Namespace)
	assert.EqualValues(t, vzV1Alpha1.Spec, vzRetrieved.Spec)
	assert.EqualValues(t, vzV1Alpha1.Status, vzRetrieved.Status)
}

func TestUpdateStatusVerrazzanoV1Alpha1NotFound(t *testing.T) {
	ctx, client, err := getTestingContextAndClient()
	assert.NoError(t, err)

	vzV1Alpha1, err := loadV1Alpha1CR(testCaseBasic)
	assert.NoError(t, err)

	// Attempt to update a nonexistent Verrazzano resource through the K8s client
	err = UpdateVerrazzanoV1Alpha1Status(ctx, client.Status(), vzV1Alpha1)

	// a NotFound error should have been returned
	assert.True(t, apierrors.IsNotFound(err), "a NotFound error was expected, but got '%v'", err)
}

// getTestingContextAndClient returns the Context and the Client used for these unit tests.
// v1beta1 is loaded into the client's scheme.
func getTestingContextAndClient() (context.Context, client.Client, error) {
	ctx := context.TODO()

	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	return ctx, client, nil
}

// createTestVZ creates a v1beta1 VZ resource through the fake client.
// The expected v1alpha1 version of that VZ resource is returned.
func createTestVZ(ctx context.Context, client client.Client) (*Verrazzano, error) {
	// create a v1beta1 Verrazzano through the K8s client
	vzStoredV1Beta1, err := loadV1Beta1(testCaseBasic)
	if err != nil {
		return nil, err
	}
	if err = client.Create(ctx, vzStoredV1Beta1); err != nil {
		return nil, err
	}

	// the expected VZ resource returned should be v1alpha1
	var vzExpected *Verrazzano
	if vzExpected, err = loadV1Alpha1CR(testCaseBasic); err != nil {
		return nil, err
	}
	return vzExpected, nil
}