// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterconfigmap

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
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

const namespace = "unit-mccm-namespace"
const crName = "unit-mccm"
const sampleMCConfigMapFile = "testdata/sample-mcconfigmap.yaml"

// TestConfigMapReconcilerSetupWithManager test the creation of the MultiClusterConfigMapReconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestConfigMapReconcilerSetupWithManager(t *testing.T) {
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

// TestReconcileCreateConfigMap tests the basic happy path of reconciling a MultiClusterConfigMap. We
// expect to write out a K8S ConfigMap
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called
// THEN expect a K8S ConfigMap to be created
func TestReconcileCreateConfigMap(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getMCConfigMap(sampleMCConfigMapFile)

	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcCompSample)

	// expect a call to fetch existing K8S ConfigMap, and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ConfigMap"}, crName))

	// expect a call to create the K8S ConfigMap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			assertConfigMapValid(assert, c, mcCompSample)
			return nil
		})

	// expect a call to update the status of the MultiClusterConfigMap
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateConfigMap tests the path of reconciling a MultiClusterConfigMap when the
// underlying K8S ConfigMap already exists i.e. update case
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called
// THEN expect a K8S ConfigMap to be updated
func TestReconcileUpdateConfigMap(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcCompSample)

	// expect a call to fetch underlying K8S ConfigMap, and return an existing component
	doExpectGetConfigMapExists(cli, mcCompSample.ObjectMeta)

	// expect a call to update the K8S ConfigMap with the new component workload data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			assertConfigMapValid(assert, c, mcCompSample)
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

// TestReconcileCreateConfigMapFailed tests the path of reconciling a MultiClusterConfigMap
// when the underlying K8S ConfigMap does not exist and fails to be created due to some error condition
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called and create underlying component fails
// THEN expect the status of the MultiClusterConfigMap to be updated with failure information
func TestReconcileCreateConfigMapFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcCompSample)

	// expect a call to fetch existing K8S ConfigMap and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ConfigMap"}, crName))

	// expect a call to create the K8S ConfigMap and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterConfigMap is updated to failed because we
	// failed the underlying K8S ConfigMap's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateConfigMapFailed tests the path of reconciling a MultiClusterConfigMap
// when the underlying K8S ConfigMap exists and fails to be updated due to some error condition
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called and update underlying component fails
// THEN expect the status of the MultiClusterConfigMap to be updated with failure information
func TestReconcileUpdateConfigMapFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcCompSample, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcCompSample)

	// expect a call to fetch existing K8S ConfigMap (simulate update case)
	doExpectGetConfigMapExists(cli, mcCompSample.ObjectMeta)

	// expect a call to update the K8S ConfigMap and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterConfigMap is updated to failed because we
	// failed the underlying K8S ConfigMap's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusters.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// doExpectGetConfigMapExists expects a call to get an K8S ConfigMap and return an "existing" one
func doExpectGetConfigMapExists(cli *mocks.MockClient, metadata metav1.ObjectMeta) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *v1.ConfigMap) error {
			existingCM, err := getExistingConfigMap()
			if err == nil {
				existingCM.DeepCopyInto(configMap)
			}
			return err
		})
}

// doExpectStatusUpdateFailed expects a call to update status of MultiClusterConfigMap to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the MultiClusterConfigMap
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the MultiClusterConfigMap
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterConfigMap{})).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterConfigMap) error {
			assertMultiClusterConfigMapStatus(assert, mcComp, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterConfigMap to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the MultiClusterConfigMap
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the MultiClusterConfigMap
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterConfigMap{})).
		DoAndReturn(func(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterConfigMap) error {
			assertMultiClusterConfigMapStatus(assert, mcComp, clustersv1alpha1.Ready, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterConfigMap adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterConfigMap, and populate the multi cluster component with given data
func doExpectGetMultiClusterConfigMap(cli *mocks.MockClient, mcCompSample clustersv1alpha1.MultiClusterConfigMap) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcCompSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComp *clustersv1alpha1.MultiClusterConfigMap) error {
			mcComp.ObjectMeta = mcCompSample.ObjectMeta
			mcComp.TypeMeta = mcCompSample.TypeMeta
			mcComp.Spec = mcCompSample.Spec
			return nil
		})
}

// assertMultiClusterConfigMapStatus asserts that the status and conditions on the MultiClusterConfigMap
// are as expected
func assertMultiClusterConfigMapStatus(assert *asserts.Assertions, mcComp *clustersv1alpha1.MultiClusterConfigMap, state clustersv1alpha1.StateType, condition clustersv1alpha1.ConditionType, conditionStatus v1.ConditionStatus) {
	assert.Equal(state, mcComp.Status.State)
	assert.Equal(1, len(mcComp.Status.Conditions))
	assert.Equal(conditionStatus, mcComp.Status.Conditions[0].Status)
	assert.Equal(condition, mcComp.Status.Conditions[0].Type)
}

// assertConfigMapValid asserts that the metadata and content of the created/updated K8S ConfigMap
// are valid
func assertConfigMapValid(assert *asserts.Assertions, cm *v1.ConfigMap, mcConfigMap clustersv1alpha1.MultiClusterConfigMap) {
	assert.Equal(namespace, cm.ObjectMeta.Namespace)
	assert.Equal(crName, cm.ObjectMeta.Name)
	assert.Equal(mcConfigMap.Spec.Template.Data, cm.Data)
	assert.Equal(mcConfigMap.Spec.Template.BinaryData, cm.BinaryData)

	// assert that the owner reference points to a MultiClusterConfigMap
	assert.Equal(1, len(cm.OwnerReferences))
	assert.Equal("MultiClusterConfigMap", cm.OwnerReferences[0].Kind)
	assert.Equal(clustersv1alpha1.GroupVersion.String(), cm.OwnerReferences[0].APIVersion)
	assert.Equal(crName, cm.OwnerReferences[0].Name)
}

// getMCConfigMap creates and returns a sample MultiClusterConfigMap used in tests
func getMCConfigMap(filename string) (clustersv1alpha1.MultiClusterConfigMap, error) {
	mcConfigMap := clustersv1alpha1.MultiClusterConfigMap{}
	sampleConfigMapFile, err := filepath.Abs(filename)
	if err != nil {
		return mcConfigMap, err
	}

	rawMcComp, err := clusters.ReadYaml2Json(sampleConfigMapFile)
	if err != nil {
		return mcConfigMap, err
	}

	err = json.Unmarshal(rawMcComp, &mcConfigMap)
	return mcConfigMap, err
}

func getExistingConfigMap() (v1.ConfigMap, error) {
	configMap := v1.ConfigMap{}
	existingConfigMapFile, err := filepath.Abs("testdata/sample-configmap-existing.yaml")
	if err != nil {
		return configMap, err
	}
	rawMcComp, err := clusters.ReadYaml2Json(existingConfigMapFile)
	if err != nil {
		return configMap, err
	}

	err = json.Unmarshal(rawMcComp, &configMap)
	return configMap, err
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
