// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"fmt"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestFetchWorkloadDefinition tests the FetchWorkloadDefinition function
func TestFetchWorkloadDefinition(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var workload unstructured.Unstructured
	var definition *oamcore.WorkloadDefinition

	// GIVEN a nil workload reference
	// WHEN an attempt is made to fetch the workload definition
	// THEN expect an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, nil)
	mocker.Finish()
	assert.Error(err)
	assert.Nil(definition)

	// GIVEN a valid workload reference
	// WHEN an attempt is made to fetch the workload definition
	// THEN the workload definition to be returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			return nil
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(definition)
	assert.Equal("containerizedworkloads.core.oam.dev", definition.Name)

	// GIVEN a valid workload reference
	// WHEN an underlying error occurs with the k8s api
	// THEN expect the error will be propagated
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			return fmt.Errorf("test-error")
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.Error(err)
	assert.Equal("test-error", err.Error())
	assert.Nil(definition)
}

// TestFetchWorkloadChildren tests the FetchWorkloadChildren function.
func TestFetchWorkloadChildren(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var workload unstructured.Unstructured
	var children []*unstructured.Unstructured

	// GIVEN a nil workload parameter
	// WHEN the workload children are fetched
	// THEN verify an error is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, nil)
	mocker.Finish()
	assert.Error(err)
	assert.Len(children, 0)

	// GIVEN a valid list of workload children
	// WHEN the a workloads children are fetched
	// THEN verify that the workload children are returned correctly.
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect a call to get the containerized workload definition and return one that populates the child resource kinds.
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			wlDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}}
			return nil
		})
	// Expect a call to list the children resources and return a list.
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
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	workload.SetNamespace("test-namespace")
	workload.SetName("test-workload-name")
	workload.SetUID("test-workload-uid")
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.NoError(err)
	assert.Len(children, 1)
	assert.Equal("test-deployment-name", children[0].GetName())

	// GIVEN a request to fetch a workload's children
	// WHEN an underlying kubernetes api error occurs
	// THEN verify that the error is propigated to the caller.
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			wlDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}}
			return nil
		})
	cli.EXPECT().
		List(gomock.Eq(ctx), gomock.Not(gomock.Nil()), gomock.Eq(client.InNamespace("test-namespace")), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, namespace client.InNamespace, labels client.MatchingLabels) error {
			return fmt.Errorf("test-error")
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	workload.SetNamespace("test-namespace")
	workload.SetName("test-workload-name")
	workload.SetUID("test-workload-uid")
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.Error(err)
	assert.Equal("test-error", err.Error())
	assert.Len(children, 0)
}
