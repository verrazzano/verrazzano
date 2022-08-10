// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustercomponent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const namespace = "unit-mccomp-namespace"
const crName = "unit-mccomp"

// TestComponentReconcilerSetupWithManager test the creation of the MultiClusterComponentReconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestComponentReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler Reconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileCreateComponent tests the basic happy path of reconciling a MultiClusterComponent. We
// expect to write out an OAM component
// GIVEN a MultiClusterComponent resource is created
// WHEN the controller Reconcile function is called
// THEN expect an OAM component to be created
func TestReconcileCreateComponent(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCComponent()

	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterComponent
	doExpectGetMultiClusterComponent(cli, mcCompSample, false)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM component, and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "core.oam.dev", Resource: "Component"}, crName))

	// expect a call to create the OAM component
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.Component, opts ...client.CreateOption) error {
			assertComponentValid(assert, c, mcCompSample)
			return nil
		})

	// expect a call to update the resource with a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, component *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			assert.True(len(component.ObjectMeta.Finalizers) == 1, "Wrong number of finalizers")
			assert.Equal(finalizerName, component.ObjectMeta.Finalizers[0], "wrong finalizer")
			return nil
		})

	// expect a call to update the status of the MultiClusterComponent
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateComponent tests the path of reconciling a MultiClusterComponent when the
// underlying OAM component already exists i.e. update case
// GIVEN a MultiClusterComponent resource is created
// WHEN the controller Reconcile function is called
// THEN expect an OAM component to be updated
func TestReconcileUpdateComponent(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	existingOAMComp, err := getExistingOAMComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterComponent
	doExpectGetMultiClusterComponent(cli, mcCompSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch underlying OAM component, and return an existing component
	doExpectGetComponentExists(cli, mcCompSample.ObjectMeta, existingOAMComp.Spec)

	// expect a call to update the OAM component with the new component workload data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.Component, opts ...client.CreateOption) error {
			assertComponentValid(assert, c, mcCompSample)
			return nil
		})

	// expect a call to update the status of the multicluster component
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateComponentFailed tests the path of reconciling a MultiClusterComponent
// when the underlying OAM component does not exist and fails to be created due to some error condition
// GIVEN a MultiClusterComponent resource is created
// WHEN the controller Reconcile function is called and create underlying component fails
// THEN expect the status of the MultiClusterComponent to be updated with failure information
func TestReconcileCreateComponentFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterComponent
	doExpectGetMultiClusterComponent(cli, mcCompSample, false)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM component and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "core.oam.dev", Resource: "Component"}, crName))

	// expect a call to create the OAM component and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.Component, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterComponent is updated to failed because we
	// failed the underlying OAM component's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestReconcileCreateComponentFailed tests the path of reconciling a MultiClusterComponent
// when the underlying OAM component exists and fails to be updated due to some error condition
// GIVEN a MultiClusterComponent resource is created
// WHEN the controller Reconcile function is called and update underlying component fails
// THEN expect the status of the MultiClusterComponent to be updated with failure information
func TestReconcileUpdateComponentFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	existingOAMComp, err := getExistingOAMComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterComponent
	doExpectGetMultiClusterComponent(cli, mcCompSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM component (simulate update case)
	doExpectGetComponentExists(cli, mcCompSample.ObjectMeta, existingOAMComp.Spec)

	// expect a call to update the OAM component and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.Component, opts ...client.UpdateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterComponent is updated to failed because we
	// failed the underlying OAM component's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestReconcilePlacementInDifferentCluster tests the path of reconciling a MultiClusterComponent which
// is placed on a cluster other than the current cluster. We expect this MultiClusterComponent to
// be ignored, and no OAM Component to be created
// GIVEN a MultiClusterComponent resource is created with a placement in different cluster
// WHEN the controller Reconcile function is called
// THEN expect that no OAM Component is created
func TestReconcilePlacementInDifferentCluster(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	statusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCComponent()
	if err != nil {
		t.Fatalf(err.Error())
	}

	mcCompSample.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// expect a call to fetch the MultiClusterComponent
	doExpectGetMultiClusterComponent(cli, mcCompSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// The effective state of the object will get updated even if it is note locally placed,
	// since it would have changed
	clusterstest.DoExpectUpdateState(t, cli, statusWriter, &mcCompSample, clustersv1alpha1.Pending)

	clusterstest.ExpectDeleteAssociatedResource(cli, &v1alpha2.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcCompSample.Name,
			Namespace: mcCompSample.Namespace,
		},
	}, types.NamespacedName{
		Namespace: mcCompSample.Namespace,
		Name:      mcCompSample.Name,
	})

	// expect a call to update the resource with no finalizers
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComponent *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			assert.True(len(mcComponent.Finalizers) == 0, "Wrong number of finalizers")
			return nil
		})

	// Expect no further action

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileResourceNotFound tests the path of reconciling a
// MultiClusterComponent resource which is non-existent when reconcile is called,
// possibly because it has been deleted.
// GIVEN a MultiClusterComponent resource has been deleted
// WHEN the controller Reconcile function is called
// THEN expect that no action is taken
func TestReconcileResourceNotFound(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// expect a call to fetch the MultiClusterComponent
	// and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: clustersv1alpha1.SchemeGroupVersion.Group, Resource: clustersv1alpha1.MultiClusterComponentResource}, crName))

	// expect no further action to be taken

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// doExpectGetComponentExists expects a call to get an OAM component and return an "existing" one
func doExpectGetComponentExists(cli *mocks.MockClient, metadata metav1.ObjectMeta, componentSpec v1alpha2.ComponentSpec) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *v1alpha2.Component) error {
			component.Spec = componentSpec
			component.ObjectMeta = metadata
			return nil
		})
}

// doExpectStatusUpdateFailed expects a call to update status of MultiClusterComponent to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterComponent
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the MultiClusterComponent
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterComponent{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcComp.Status,
				clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterComponent to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterComponent
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the MultiClusterComponent
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterComponent{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcComp.Status,
				clustersv1alpha1.Succeeded, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterComponent adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterComponent, and populate the multi cluster component with given data
func doExpectGetMultiClusterComponent(cli *mocks.MockClient, mcCompSample clustersv1alpha1.MultiClusterComponent, addFinalizer bool) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcCompSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComp *clustersv1alpha1.MultiClusterComponent) error {
			mcComp.ObjectMeta = mcCompSample.ObjectMeta
			mcComp.TypeMeta = mcCompSample.TypeMeta
			mcComp.Spec = mcCompSample.Spec
			if addFinalizer {
				mcComp.Finalizers = append(mcComp.Finalizers, finalizerName)
			}
			return nil
		})
}

// assertComponentValid asserts that the metadata and content of the created/updated OAM component
// are valid
func assertComponentValid(assert *asserts.Assertions, c *v1alpha2.Component, mcComp clustersv1alpha1.MultiClusterComponent) {
	assert.Equal(namespace, c.ObjectMeta.Namespace)
	assert.Equal(crName, c.ObjectMeta.Name)
	assert.Equal(mcComp.Spec.Template.Spec, c.Spec)

	// assert that the component is labeled verrazzano-managed=true since it was created by Verrazzano
	assert.NotNil(c.Labels)
	assert.Equal(constants.LabelVerrazzanoManagedDefault, c.Labels[vzconst.VerrazzanoManagedLabelKey])

	// assert some fields on the component spec (e.g. in the case of update, these fields should
	// be different from the mock pre existing OAM component)
	expectedContainerizedWorkload, err := clusterstest.ReadContainerizedWorkload(mcComp.Spec.Template.Spec.Workload)
	assert.Nil(err)
	actualContainerizedWorkload, err := clusterstest.ReadContainerizedWorkload(c.Spec.Workload)
	assert.Nil(err)
	assert.Equal(expectedContainerizedWorkload.Spec.Containers[0].Name, actualContainerizedWorkload.Spec.Containers[0].Name)
	assert.Equal(expectedContainerizedWorkload.Name, actualContainerizedWorkload.Name)

}

// getSampleMCComponent creates and returns a sample MultiClusterComponent used in tests
func getSampleMCComponent() (clustersv1alpha1.MultiClusterComponent, error) {
	mcComp := clustersv1alpha1.MultiClusterComponent{}
	sampleComponentFile, err := filepath.Abs("testdata/hello-multiclustercomponent.yaml")
	if err != nil {
		return mcComp, err
	}

	rawMcComp, err := clusterstest.ReadYaml2Json(sampleComponentFile)
	if err != nil {
		return mcComp, err
	}

	err = json.Unmarshal(rawMcComp, &mcComp)
	return mcComp, err
}

func getExistingOAMComponent() (v1alpha2.Component, error) {
	oamComp := v1alpha2.Component{}
	existingComponentFile, err := filepath.Abs("testdata/hello-oam-comp-existing.yaml")
	if err != nil {
		return oamComp, err
	}
	rawMcComp, err := clusterstest.ReadYaml2Json(existingComponentFile)
	if err != nil {
		return oamComp, err
	}

	err = json.Unmarshal(rawMcComp, &oamComp)
	return oamComp, err
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    zap.S().With("test"),
		Scheme: clusters.NewScheme(),
	}
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := clusterstest.NewRequest(vzconst.KubeSystem, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}
