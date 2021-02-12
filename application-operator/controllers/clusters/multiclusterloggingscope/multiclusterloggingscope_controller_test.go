// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterloggingscope

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const namespace = "unit-mclogscope-namespace"
const crName = "unit-mclogscope"

// TestLoggingScopeReconcilerSetupWithManager test the creation of the MultiClusterLoggingScopeReconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestLoggingScopeReconcilerSetupWithManager(t *testing.T) {
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
	clustersv1alpha1.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileCreateLoggingScope tests the basic happy path of reconciling a MultiClusterLoggingScope. We
// expect to write out a LoggingScope
// GIVEN a MultiClusterLoggingScope resource is created
// WHEN the controller Reconcile function is called
// THEN expect a LoggingScope to be created
func TestReconcileCreateLoggingScope(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCLoggingScope()

	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterLoggingScope
	doExpectGetMultiClusterLoggingScope(cli, mcCompSample)

	// expect a call to fetch existing LoggingScope, and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "LoggingScope"}, crName))

	// expect a call to create the LoggingScope
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha1.LoggingScope, opts ...client.CreateOption) error {
			assertLoggingScopeValid(assert, c, mcCompSample)
			return nil
		})

	// expect a call to update the status of the MultiClusterLoggingScope
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateLoggingScope tests the path of reconciling a MultiClusterLoggingScope when the
// underlying LoggingScope already exists i.e. update case
// GIVEN a MultiClusterLoggingScope resource is created
// WHEN the controller Reconcile function is called
// THEN expect a LoggingScope to be updated
func TestReconcileUpdateLoggingScope(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCLoggingScope()
	if err != nil {
		t.Fatalf(err.Error())
	}

	existingOAMComp, err := getExistingLoggingScope()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterLoggingScope
	doExpectGetMultiClusterLoggingScope(cli, mcCompSample)

	// expect a call to fetch underlying LoggingScope, and return an existing component
	doExpectGetLoggingScopeExists(cli, mcCompSample.ObjectMeta, existingOAMComp.Spec)

	// expect a call to update the LoggingScope with the new component workload data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha1.LoggingScope, opts ...client.CreateOption) error {
			assertLoggingScopeValid(assert, c, mcCompSample)
			return nil
		})

	// expect a call to update the status of the multicluster component
	cli.EXPECT().Status().Return(mockStatusWriter)

	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&mcCompSample)).
		Return(nil)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateLoggingScopeFailed tests the path of reconciling a MultiClusterLoggingScope
// when the underlying LoggingScope does not exist and fails to be created due to some error condition
// GIVEN a MultiClusterLoggingScope resource is created
// WHEN the controller Reconcile function is called and create underlying component fails
// THEN expect the status of the MultiClusterLoggingScope to be updated with failure information
func TestReconcileCreateLoggingScopeFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCLoggingScope()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterLoggingScope
	doExpectGetMultiClusterLoggingScope(cli, mcCompSample)

	// expect a call to fetch existing LoggingScope and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "LoggingScope"}, crName))

	// expect a call to create the LoggingScope and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha1.LoggingScope, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterLoggingScope is updated to failed because we
	// failed the underlying LoggingScope's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateLoggingScopeFailed tests the path of reconciling a MultiClusterLoggingScope
// when the underlying LoggingScope exists and fails to be updated due to some error condition
// GIVEN a MultiClusterLoggingScope resource is created
// WHEN the controller Reconcile function is called and update underlying component fails
// THEN expect the status of the MultiClusterLoggingScope to be updated with failure information
func TestReconcileUpdateLoggingScopeFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getSampleMCLoggingScope()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterLoggingScope
	doExpectGetMultiClusterLoggingScope(cli, mcCompSample)

	// expect a call to fetch existing LoggingScope (simulate update case)
	doExpectGetLoggingScopeExists(cli, mcCompSample.ObjectMeta, mcCompSample.Spec.Template.Spec)

	// expect a call to update the LoggingScope and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha1.LoggingScope, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterLoggingScope is updated to failed because we
	// failed the underlying LoggingScope's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// doExpectGetLoggingScopeExists expects a call to get a LoggingScope and return an "existing" one
func doExpectGetLoggingScopeExists(cli *mocks.MockClient, metadata metav1.ObjectMeta, componentSpec v1alpha1.LoggingScopeSpec) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *v1alpha1.LoggingScope) error {
			component.Spec = componentSpec
			component.ObjectMeta = metadata
			return nil
		})
}

// doExpectStatusUpdateFailed expects a call to update status of MultiClusterLoggingScope to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the MultiClusterLoggingScope
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the MultiClusterLoggingScope
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterLoggingScope{})).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterLoggingScope) error {
			assertMultiClusterLoggingScopeStatus(assert, mcComp, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterLoggingScope to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the MultiClusterLoggingScope
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the MultiClusterLoggingScope
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterLoggingScope{})).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterLoggingScope) error {
			assertMultiClusterLoggingScopeStatus(assert, mcComp, clustersv1alpha1.Ready, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterLoggingScope adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterLoggingScope, and populate the multi cluster component with given data
func doExpectGetMultiClusterLoggingScope(cli *mocks.MockClient, mcCompSample clustersv1alpha1.MultiClusterLoggingScope) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcCompSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComp *clustersv1alpha1.MultiClusterLoggingScope) error {
			mcComp.ObjectMeta = mcCompSample.ObjectMeta
			mcComp.TypeMeta = mcCompSample.TypeMeta
			mcComp.Spec = mcCompSample.Spec
			return nil
		})
}

// assertMultiClusterLoggingScopeStatus asserts that the status and conditions on the MultiClusterLoggingScope
// are as expected
func assertMultiClusterLoggingScopeStatus(assert *asserts.Assertions, mcComp *clustersv1alpha1.MultiClusterLoggingScope, state clustersv1alpha1.StateType, condition clustersv1alpha1.ConditionType, conditionStatus v1.ConditionStatus) {
	assert.Equal(state, mcComp.Status.State)
	assert.Equal(1, len(mcComp.Status.Conditions))
	assert.Equal(conditionStatus, mcComp.Status.Conditions[0].Status)
	assert.Equal(condition, mcComp.Status.Conditions[0].Type)
}

// assertLoggingScopeValid asserts that the metadata and content of the created/updated LoggingScope
// are valid
func assertLoggingScopeValid(assert *asserts.Assertions, logScope *v1alpha1.LoggingScope, mcLogScope clustersv1alpha1.MultiClusterLoggingScope) {
	assert.Equal(namespace, logScope.ObjectMeta.Namespace)
	assert.Equal(crName, logScope.ObjectMeta.Name)
	assert.Equal(mcLogScope.Spec.Template.Spec, logScope.Spec)

	// assert fields on the LoggingScope spec (e.g. in the case of update, these fields should
	// be different from the mock pre existing LoggingScope)
	assert.Equal(mcLogScope.Spec.Template.Spec.ElasticSearchHost, logScope.Spec.ElasticSearchHost)
	assert.Equal(mcLogScope.Spec.Template.Spec.FluentdImage, logScope.Spec.FluentdImage)
	assert.Equal(mcLogScope.Spec.Template.Spec.ElasticSearchPort, logScope.Spec.ElasticSearchPort)
	assert.Equal(mcLogScope.Spec.Template.Spec.SecretName, logScope.Spec.SecretName)

	// assert that the owner reference points to a MultiClusterLoggingScope
	assert.Equal(1, len(logScope.OwnerReferences))
	assert.Equal("MultiClusterLoggingScope", logScope.OwnerReferences[0].Kind)
	assert.Equal(clustersv1alpha1.GroupVersion.String(), logScope.OwnerReferences[0].APIVersion)
	assert.Equal(crName, logScope.OwnerReferences[0].Name)
}

// getSampleMCLoggingScope creates and returns a sample MultiClusterLoggingScope used in tests
func getSampleMCLoggingScope() (clustersv1alpha1.MultiClusterLoggingScope, error) {
	mcComp := clustersv1alpha1.MultiClusterLoggingScope{}
	sampleLoggingScopeFile, err := filepath.Abs("testdata/sample-multiclusterloggingscope.yaml")
	if err != nil {
		return mcComp, err
	}

	rawMcComp, err := clusters.ReadYaml2Json(sampleLoggingScopeFile)
	if err != nil {
		return mcComp, err
	}

	err = json.Unmarshal(rawMcComp, &mcComp)
	return mcComp, err
}

func getExistingLoggingScope() (v1alpha1.LoggingScope, error) {
	oamComp := v1alpha1.LoggingScope{}
	existingLoggingScopeFile, err := filepath.Abs("testdata/loggingscope-existing.yaml")
	if err != nil {
		return oamComp, err
	}
	rawMcComp, err := clusters.ReadYaml2Json(existingLoggingScopeFile)
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
		Log:    ctrl.Log.WithName("test"),
		Scheme: clusters.NewScheme(),
	}
}
