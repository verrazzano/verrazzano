// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	oamrt "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestFetchTrait tests various usages of FetchTrait
func TestFetchTrait(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var trait *vzapi.MetricsTrait
	var err error
	var name types.NamespacedName

	// GIVEN a name for a trait that does exists
	// WHEN the trait is fetched
	// THEN verify that the returned trait has correct content
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	name = types.NamespacedName{Namespace: "test-namespace", Name: "test-name"}
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(name), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, trait *vzapi.MetricsTrait) error {
			trait.Name = "test-name"
			return nil
		})
	trait, err = FetchTrait(context.TODO(), cli, zap.S(), name)
	mocker.Finish()
	assert.NoError(err)
	assert.Equal("test-name", trait.Name)

	// GIVEN a name for a trait that does not exist
	// WHEN the trait is fetched
	// THEN verify that both the returned trait and error are nil
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	name = types.NamespacedName{Namespace: "test-namespace", Name: "test-name"}
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(name), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, trait *vzapi.MetricsTrait) error {
			return k8serrors.NewNotFound(schema.GroupResource{Group: trait.APIVersion, Resource: trait.Kind}, key.Name)
		})
	trait, err = FetchTrait(context.TODO(), cli, zap.S(), name)
	mocker.Finish()
	assert.Nil(trait)
	assert.NoError(err)

	// GIVEN a name for a trait that should exist
	// WHEN the trait is fetched and there is an underlying error
	// THEN verify that the error is propagated
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(name), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, trait *vzapi.MetricsTrait) error {
			return fmt.Errorf("test-error")
		})
	name = types.NamespacedName{Namespace: "test-namespace", Name: "test-name"}
	trait, err = FetchTrait(context.TODO(), cli, zap.S(), name)
	mocker.Finish()
	assert.Nil(trait)
	assert.Error(err)
	assert.Equal("test-error", err.Error())
}

// TestFetchWorkloadFromTrait tests various usages of FetchWorkloadFromTrait
func TestFetchWorkloadFromTrait(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var trait *vzapi.IngressTrait
	var err error
	var uns *unstructured.Unstructured

	// GIVEN a trait with a reference to a workload that can be found
	// WHEN the workload is fetched via the trait
	// THEN verify the workload content is correct
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	trait = &vzapi.IngressTrait{
		TypeMeta:   metav1.TypeMeta{Kind: "IngressTrait", APIVersion: "oam.verrazzano.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-trait-name", Namespace: "test-trait-namespace"},
		Spec: vzapi.IngressTraitSpec{WorkloadReference: oamrt.TypedReference{
			APIVersion: "core.oam.dev/v1alpha2", Kind: "ContainerizedWorkload", Name: "test-workload-name"}}}
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-trait-namespace", Name: "test-workload-name"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			obj.SetNamespace(key.Namespace)
			obj.SetName(key.Name)
			return nil
		})
	uns, err = FetchWorkloadFromTrait(ctx, cli, vzlog.DefaultLogger(), trait)
	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(uns)
	assert.Equal("test-trait-namespace", uns.GetNamespace())
	assert.Equal("test-workload-name", uns.GetName())

	// GIVEN a trait with a reference to a workload
	// WHEN a failure occurs attempting to fetch the trait's workload
	// THEN verify the error is propagated
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	trait = &vzapi.IngressTrait{
		TypeMeta:   metav1.TypeMeta{Kind: "IngressTrait", APIVersion: "oam.verrazzano.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-trait-name", Namespace: "test-trait-namespace"},
		Spec: vzapi.IngressTraitSpec{WorkloadReference: oamrt.TypedReference{
			APIVersion: "core.oam.dev/v1alpha2", Kind: "ContainerizedWorkload", Name: "test-workload-name"}}}
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-trait-namespace", Name: "test-workload-name"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			return fmt.Errorf("test-error")
		})
	uns, err = FetchWorkloadFromTrait(ctx, cli, vzlog.DefaultLogger(), trait)
	mocker.Finish()
	assert.Nil(uns)
	assert.Error(err)
	assert.Equal("test-error", err.Error())

	// GIVEN a trait with a reference to a Verrazzano workload type
	// WHEN the workload is fetched via the trait
	// THEN verify that the contained workload is unwrapped and returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	workloadAPIVersion := "oam.verrazzano.io/v1alpha1"
	workloadKind := "VerrazzanoCoherenceWorkload"

	containedAPIVersion := "coherence.oracle.com/v1"
	containedKind := "Coherence"
	containedName := "unit-test-resource"

	containedResource := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	trait = &vzapi.IngressTrait{
		TypeMeta:   metav1.TypeMeta{Kind: "IngressTrait", APIVersion: "oam.verrazzano.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-trait-name", Namespace: "test-trait-namespace"},
		Spec: vzapi.IngressTraitSpec{WorkloadReference: oamrt.TypedReference{
			APIVersion: workloadAPIVersion, Kind: workloadKind, Name: "test-workload-name"}}}

	// expect a call to fetch the referenced workload
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-trait-namespace", Name: "test-workload-name"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			obj.SetNamespace(key.Namespace)
			obj.SetName(key.Name)
			obj.SetAPIVersion(workloadAPIVersion)
			obj.SetKind(workloadKind)
			unstructured.SetNestedMap(obj.Object, containedResource, "spec", "template")
			return nil
		})
	// expect a call to fetch the contained workload that is wrapped by the Verrazzano workload
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-trait-namespace", Name: containedName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			obj.SetUnstructuredContent(containedResource)
			obj.SetAPIVersion(containedAPIVersion)
			obj.SetKind(containedKind)
			return nil
		})

	uns, err = FetchWorkloadFromTrait(ctx, cli, vzlog.DefaultLogger(), trait)

	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(uns)
	assert.Equal(containedAPIVersion, uns.GetAPIVersion())
	assert.Equal(containedKind, uns.GetKind())
	assert.Equal(containedName, uns.GetName())

	// GIVEN a trait with a reference to a VerrazzanoHelidonWorkload that can be found
	// WHEN the workload is fetched via the trait
	// THEN verify the workload content is correct
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	trait = &vzapi.IngressTrait{
		TypeMeta:   metav1.TypeMeta{Kind: "IngressTrait", APIVersion: "oam.verrazzano.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-trait-name", Namespace: "test-trait-namespace"},
		Spec: vzapi.IngressTraitSpec{WorkloadReference: oamrt.TypedReference{
			APIVersion: "oam.verrazzano.io/v1alpha1", Kind: "VerrazzanoHelidonWorkload", Name: "test-helidon-workload"}}}
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "test-trait-namespace", Name: "test-helidon-workload"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			obj.SetNamespace(key.Namespace)
			obj.SetName(key.Name)
			return nil
		})
	uns, err = FetchWorkloadFromTrait(ctx, cli, vzlog.DefaultLogger(), trait)
	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(uns)

	assert.Equal("test-trait-namespace", uns.GetNamespace())
	assert.Equal("test-helidon-workload", uns.GetName())
}

// TestFetchWorkloadResource tests various usages of FetchWorkloadResource
func TestFetchWorkloadResource(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var uns *unstructured.Unstructured

	// GIVEN a non Verrazzano specific workload
	// WHEN the workload resource is fetched
	// THEN verify the workload is returned as is
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	uns = &unstructured.Unstructured{}
	containerizedWorkloadName := "container-workload"
	containerizedWorkloadNamespace := "default"
	containerizedWorkloadAPIVersion := "core.oam.dev/v1alpha2"
	containerizedWorkloadKind := "ContainerizedWorkload"
	uns.SetNamespace(containerizedWorkloadNamespace)
	uns.SetName(containerizedWorkloadName)
	uns.SetAPIVersion(containerizedWorkloadAPIVersion)
	uns.SetKind(containerizedWorkloadKind)
	uns, err = FetchWorkloadResource(ctx, cli, vzlog.DefaultLogger(), uns)
	assert.NoError(err)
	assert.Equal(containerizedWorkloadNamespace, uns.GetNamespace())
	assert.Equal(containerizedWorkloadName, uns.GetName())
	assert.Equal(containerizedWorkloadAPIVersion, uns.GetAPIVersion())
	assert.Equal(containerizedWorkloadKind, uns.GetKind())

	// GIVEN a Verrazzano specific workload
	// WHEN the workload resource is fetched
	// THEN verify the contained workload is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	uns = &unstructured.Unstructured{}
	workloadName := "coherence-workload"
	workloadNamespace := "default"
	workloadAPIVersion := "oam.verrazzano.io/v1alpha1"
	workloadKind := "VerrazzanoCoherenceWorkload"

	containedAPIVersion := "coherence.oracle.com/v1"
	containedKind := "Coherence"
	containedName := "unit-test-resource"
	containedNamespace := "default"

	containedResource := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	uns.SetNamespace(workloadNamespace)
	uns.SetName(workloadName)
	uns.SetAPIVersion(workloadAPIVersion)
	uns.SetKind(workloadKind)
	unstructured.SetNestedMap(uns.Object, containedResource, "spec", "template")
	// expect a call to fetch the contained workload that is wrapped by the Verrazzano workload
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: containedNamespace, Name: containedName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			obj.SetUnstructuredContent(containedResource)
			obj.SetAPIVersion(containedAPIVersion)
			obj.SetKind(containedKind)
			obj.SetNamespace(containedNamespace)
			return nil
		})

	uns, err = FetchWorkloadResource(ctx, cli, vzlog.DefaultLogger(), uns)
	assert.NoError(err)
	assert.Equal(containedNamespace, uns.GetNamespace())
	assert.Equal(containedName, uns.GetName())
	assert.Equal(containedAPIVersion, uns.GetAPIVersion())
	assert.Equal(containedKind, uns.GetKind())

	// GIVEN a Verrazzano specific workload
	// WHEN a failure occurs attempting to fetch the workload resource
	// THEN verify the error is propagated
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	uns = &unstructured.Unstructured{}
	workloadName = "coherence-workload"
	workloadNamespace = "default"
	workloadAPIVersion = "oam.verrazzano.io/v1alpha1"
	workloadKind = "VerrazzanoCoherenceWorkload"

	containedAPIVersion = "coherence.oracle.com/v1"
	containedKind = "Coherence"
	containedName = "unit-test-resource"
	containedNamespace = "default"

	containedResource = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	uns.SetNamespace(workloadNamespace)
	uns.SetName(workloadName)
	uns.SetAPIVersion(workloadAPIVersion)
	uns.SetKind(workloadKind)
	unstructured.SetNestedMap(uns.Object, containedResource, "spec", "template")
	// expect a call to fetch the contained workload that is wrapped by the Verrazzano workload and return error
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: containedNamespace, Name: containedName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj *unstructured.Unstructured) error {
			return fmt.Errorf("test-error")
		})

	uns, err = FetchWorkloadResource(ctx, cli, vzlog.DefaultLogger(), uns)
	mocker.Finish()
	assert.Nil(uns)
	assert.Error(err)
	assert.Equal("test-error", err.Error())
}

// TestIsWeblogicWorkloadKind tests the IsWeblogicWorkloadKind function
// GIVEN a trait with a WorkloadReference
// WHEN a call to IsWeblogicWorkloadKind is made
// THEN return true if the trait references a VerrazzanoWebLogic workload kind, false otherwise
func TestIsWeblogicWorkloadKind(t *testing.T) {
	assert := asserts.New(t)

	trait := &vzapi.IngressTrait{}
	assert.False(IsWeblogicWorkloadKind(trait))

	trait = &vzapi.IngressTrait{
		Spec: vzapi.IngressTraitSpec{
			WorkloadReference: oamrt.TypedReference{Kind: "VerrazzanoCoherenceWorkload"},
		},
	}
	assert.False(IsWeblogicWorkloadKind(trait))

	trait = &vzapi.IngressTrait{
		Spec: vzapi.IngressTraitSpec{
			WorkloadReference: oamrt.TypedReference{Kind: "VerrazzanoWebLogicWorkload"},
		},
	}
	assert.True(IsWeblogicWorkloadKind(trait))
}
