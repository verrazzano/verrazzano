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
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
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

	mcConfigMap, err := getMCConfigMap(sampleMCConfigMapFile)

	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcConfigMap)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing K8S ConfigMap, and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ConfigMap"}, crName))

	// expect a call to create the K8S ConfigMap
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			assertConfigMapValid(assert, c, mcConfigMap)
			return nil
		})

	// expect a call to update the status of the MultiClusterConfigMap
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
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

	mcConfigMap, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcConfigMap)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch underlying K8S ConfigMap, and return an existing ConfigMap
	doExpectGetConfigMapExists(cli, mcConfigMap.ObjectMeta)

	// expect a call to update the K8S ConfigMap with the new ConfigMap data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1.ConfigMap, opts ...client.CreateOption) error {
			assertConfigMapValid(assert, c, mcConfigMap)
			return nil
		})

	// expect a call to update the status of the multicluster ConfigMap\
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateConfigMapFailed tests the path of reconciling a MultiClusterConfigMap
// when the underlying K8S ConfigMap does not exist and fails to be created due to some error condition
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called and create underlying ConfigMap fails
// THEN expect the status of the MultiClusterConfigMap to be updated with failure information
func TestReconcileCreateConfigMapFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcConfigMap, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcConfigMap)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing K8S ConfigMap and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ConfigMap"}, crName))

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
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateConfigMapFailed tests the path of reconciling a MultiClusterConfigMap
// when the underlying K8S ConfigMap exists and fails to be updated due to some error condition
// GIVEN a MultiClusterConfigMap resource is created
// WHEN the controller Reconcile function is called and update underlying ConfigMap fails
// THEN expect the status of the MultiClusterConfigMap to be updated with failure information
func TestReconcileUpdateConfigMapFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcConfigMap, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcConfigMap)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing K8S ConfigMap (simulate update case)
	doExpectGetConfigMapExists(cli, mcConfigMap.ObjectMeta)

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
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcilePlacementInDifferentCluster tests the path of reconciling a MultiClusterConfigMap which
// is placed on a cluster other than the current cluster. We expect this MultiClusterConfigMap to
// be ignored, and no K8S ConfigMap to be created
// GIVEN a MultiClusterConfigMap resource is created with a placement in different cluster
// WHEN the controller Reconcile function is called
// THEN expect that no K8S ConfigMap is created
func TestReconcilePlacementInDifferentCluster(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	mcConfigMap, err := getMCConfigMap(sampleMCConfigMapFile)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mcConfigMap.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// expect a call to fetch the MultiClusterConfigMap
	doExpectGetMultiClusterConfigMap(cli, mcConfigMap)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// Expect no further action

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileResourceNotFound tests the path of reconciling a
// MultiClusterConfigMap resource which is non-existent when reconcile is called,
// possibly because it has been deleted.
// GIVEN a MultiClusterConfigMap resource has been deleted
// WHEN the controller Reconcile function is called
// THEN expect that no action is taken
func TestReconcileResourceNotFound(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// expect a call to fetch the MultiClusterConfigMap
	// and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "MultiClusterConfigMap"}, crName))

	// expect no further action to be taken

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
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
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterConfigMap
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the MultiClusterConfigMap
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterConfigMap{})).
		DoAndReturn(func(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcConfigMap.Status, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterConfigMap to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterConfigMap
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the MultiClusterConfigMap
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterConfigMap{})).
		DoAndReturn(func(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcConfigMap.Status, clustersv1alpha1.Ready, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterConfigMap adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterConfigMap, and populate the multi cluster ConfigMap with given data
func doExpectGetMultiClusterConfigMap(cli *mocks.MockClient, mcConfigMapSample clustersv1alpha1.MultiClusterConfigMap) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcConfigMapSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
			mcConfigMap.ObjectMeta = mcConfigMapSample.ObjectMeta
			mcConfigMap.TypeMeta = mcConfigMapSample.TypeMeta
			mcConfigMap.Spec = mcConfigMapSample.Spec
			return nil
		})
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

	rawResource, err := clusterstest.ReadYaml2Json(sampleConfigMapFile)
	if err != nil {
		return mcConfigMap, err
	}

	err = json.Unmarshal(rawResource, &mcConfigMap)
	return mcConfigMap, err
}

func getExistingConfigMap() (v1.ConfigMap, error) {
	configMap := v1.ConfigMap{}
	existingConfigMapFile, err := filepath.Abs("testdata/sample-configmap-existing.yaml")
	if err != nil {
		return configMap, err
	}
	rawResource, err := clusterstest.ReadYaml2Json(existingConfigMapFile)
	if err != nil {
		return configMap, err
	}

	err = json.Unmarshal(rawResource, &configMap)
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
