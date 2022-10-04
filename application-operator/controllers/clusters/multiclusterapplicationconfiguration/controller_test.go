// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterapplicationconfiguration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const namespace = "unit-mcappconfig-namespace"
const crName = "unit-mcappconfig"

// TestAppConfigReconcilerSetupWithManager test the creation of the MultiCluster app config Reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestAppConfigReconcilerSetupWithManager(t *testing.T) {
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

// TestReconcileCreateAppConfig tests the basic happy path of reconciling a
// MultiClusterApplicationConfiguration. We expect to write out an OAM app config
// GIVEN a MultiClusterApplicationConfiguration resource is created
// WHEN the controller Reconcile function is called
// THEN expect an OAM app config to be created
func TestReconcileCreateAppConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcAppConfigSample, err := getSampleMCAppConfig()

	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterApplicationConfiguration
	doExpectGetMultiClusterAppConfig(cli, mcAppConfigSample, false)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM app config, and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "core.oam.dev", Resource: "ApplicationConfiguration"}, crName))

	// expect a call to create the OAM app config
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.ApplicationConfiguration, opts ...client.CreateOption) error {
			assertAppConfigValid(assert, c, mcAppConfigSample)
			return nil
		})

	// expect a call to update the resource with a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...client.UpdateOption) error {
			assert.True(len(appConfig.ObjectMeta.Finalizers) == 1, "Wrong number of finalizers")
			assert.Equal(finalizerName, appConfig.ObjectMeta.Finalizers[0], "wrong finalizer")
			return nil
		})

	// expect a call to update the status of the MultiClusterApplicationConfiguration
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateAppConfig tests the path of reconciling a MultiClusterApplicationConfiguration
// when the underlying OAM app config already exists i.e. update case
// GIVEN a MultiClusterApplicationConfiguration resource is created
// WHEN the controller Reconcile function is called
// THEN expect an OAM app config to be updated
func TestReconcileUpdateAppConfig(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcAppConfigSample, err := getSampleMCAppConfig()
	if err != nil {
		t.Fatalf(err.Error())
	}

	existingOAMAppConfig, err := getExistingOAMAppConfig()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterApplicationConfiguration
	doExpectGetMultiClusterAppConfig(cli, mcAppConfigSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch underlying OAM app config, and return an existing one
	doExpectGetAppConfigExists(cli, mcAppConfigSample.ObjectMeta, existingOAMAppConfig.Spec)

	// expect a call to update the OAM app config with the new app config data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, app *v1alpha2.ApplicationConfiguration, opts ...client.CreateOption) error {
			assertAppConfigValid(assert, app, mcAppConfigSample)
			return nil
		})

	// expect a call to update the status of the multicluster app config
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateAppConfigFailed tests the path of reconciling a
// MultiClusterApplicationConfiguration when the underlying OAM app config does not exist and
// fails to be created due to some error condition
// GIVEN a MultiClusterApplicationConfiguration resource is created
// WHEN the controller Reconcile function is called and create underlying app config fails
// THEN expect the status of the MultiClusterApplicationConfiguration to be updated with failure
func TestReconcileCreateAppConfigFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcAppConfigSample, err := getSampleMCAppConfig()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterApplicationConfiguration
	doExpectGetMultiClusterAppConfig(cli, mcAppConfigSample, false)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM app config and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "core.oam.dev", Resource: "ApplicationConfiguration"}, crName))

	// expect a call to create the OAM app config and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.ApplicationConfiguration, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterApplicationConfiguration is updated to failed because we
	// failed the underlying OAM app config's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestReconcileUpdateAppConfigFailed tests the path of reconciling a
// MultiClusterApplicationConfiguration when the underlying OAM app config exists and fails to be
// updated due to some error condition
// GIVEN a MultiClusterApplicationConfiguration resource is created
// WHEN the controller Reconcile function is called and update underlying app config fails
// THEN expect the status of the MultiClusterApplicationConfiguration to be updated with
// failure information
func TestReconcileUpdateAppConfigFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	mcAppConfigSample, err := getSampleMCAppConfig()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// expect a call to fetch the MultiClusterApplicationConfiguration
	doExpectGetMultiClusterAppConfig(cli, mcAppConfigSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing OAM app config (simulate update case)
	doExpectGetAppConfigExists(cli, mcAppConfigSample.ObjectMeta, mcAppConfigSample.Spec.Template.Spec)

	// expect a call to update the OAM app config and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *v1alpha2.ApplicationConfiguration, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterApplicationConfiguration is updated to failed because we
	// failed the underlying OAM app config's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestReconcileResourceNotFound tests the path of reconciling a
// MultiClusterApplicationConfiguration resource which is non-existent when reconcile is called,
// possibly because it has been deleted.
// GIVEN a MultiClusterApplicationConfiguration resource has been deleted
// WHEN the controller Reconcile function is called
// THEN expect that no action is taken
func TestReconcileResourceNotFound(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// expect a call to fetch the MultiClusterApplicationConfiguration
	// and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: clustersv1alpha1.SchemeGroupVersion.Group, Resource: clustersv1alpha1.MultiClusterAppConfigResource}, crName))

	// expect no further action to be taken

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcilePlacementInDifferentCluster tests the path of reconciling a
// MultiClusterApplicationConfiguration which is placed on a cluster other than the current cluster.
// We expect this MultiClusterApplicationConfiguration to be ignored, i.e. no OAM app config created
// GIVEN a MultiClusterApplicationConfiguration resource is created with a placement in different cluster
// WHEN the controller Reconcile function is called
// THEN expect that no OAM app config is created
func TestReconcilePlacementInDifferentCluster(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	statusWriter := mocks.NewMockStatusWriter(mocker)

	mcAppConfigSample, err := getSampleMCAppConfig()
	if err != nil {
		t.Fatalf(err.Error())
	}

	mcAppConfigSample.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// expect a call to fetch the MultiClusterApplicationConfiguration
	doExpectGetMultiClusterAppConfig(cli, mcAppConfigSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// The effective state of the object will get updated even if it is note locally placed,
	// since it would have changed
	clusterstest.DoExpectUpdateState(t, cli, statusWriter, &mcAppConfigSample, clustersv1alpha1.Pending)

	clusterstest.ExpectDeleteAssociatedResource(cli, &v1alpha2.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcAppConfigSample.Name,
			Namespace: mcAppConfigSample.Namespace,
		},
	}, types.NamespacedName{
		Namespace: mcAppConfigSample.Namespace,
		Name:      mcAppConfigSample.Name,
	})
	// Expect no further action

	// expect a call to update the resource with no finalizers
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...client.UpdateOption) error {
			assert.True(len(mcAppConfig.Finalizers) == 0, "Wrong number of finalizers")
			return nil
		})

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// doExpectGetAppConfigExists expects a call to get an OAM app config and return an "existing" one
func doExpectGetAppConfigExists(cli *mocks.MockClient, metadata metav1.ObjectMeta, appConfigSpec v1alpha2.ApplicationConfigurationSpec) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *v1alpha2.ApplicationConfiguration) error {
			appConfig.Spec = appConfigSpec
			appConfig.ObjectMeta = metadata
			return nil
		})
}

// doExpectStatusUpdateFailed expects a call to update status of
// MultiClusterApplicationConfiguration to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterApplicationConfiguration
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the MultiClusterApplicationConfiguration
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterApplicationConfiguration{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, options ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcAppConfig.Status, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of
// MultiClusterApplicationConfiguration to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the MultiClusterApplicationConfiguration
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the MultiClusterApplicationConfiguration
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterApplicationConfiguration{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, options ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcAppConfig.Status, clustersv1alpha1.Succeeded, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterAppConfig adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterApplicationConfiguration, and populate it with given sample data
func doExpectGetMultiClusterAppConfig(cli *mocks.MockClient, mcAppConfigSample clustersv1alpha1.MultiClusterApplicationConfiguration, addFinalizer bool) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcAppConfigSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) error {
			mcAppConfig.ObjectMeta = mcAppConfigSample.ObjectMeta
			mcAppConfig.TypeMeta = mcAppConfigSample.TypeMeta
			mcAppConfig.Spec = mcAppConfigSample.Spec
			if addFinalizer {
				mcAppConfig.Finalizers = append(mcAppConfigSample.Finalizers, finalizerName)
			}
			return nil
		})
}

// assertAppConfigValid asserts that the metadata and content of the created/updated OAM app config
// are valid
func assertAppConfigValid(assert *asserts.Assertions, app *v1alpha2.ApplicationConfiguration, mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration) {
	assert.Equal(namespace, app.ObjectMeta.Namespace)
	assert.Equal(crName, app.ObjectMeta.Name)
	assert.Equal(mcAppConfig.Spec.Template.Spec, app.Spec)

	// assert that the app config is labeled verrazzano-managed=true since it was created by Verrazzano
	assert.NotNil(app.Labels)
	assert.Equal(constants.LabelVerrazzanoManagedDefault, app.Labels[vzconst.VerrazzanoManagedLabelKey])

	// assert some fields on the app config spec (e.g. in the case of update, these fields should
	// be different from the mock pre existing OAM app config)
	assert.Equal(len(mcAppConfig.Spec.Template.Spec.Components), len(app.Spec.Components))
	for i, comp := range mcAppConfig.Spec.Template.Spec.Components {
		assert.Equal(comp.ComponentName, app.Spec.Components[i].ComponentName)
		assert.Equal(comp.ParameterValues, app.Spec.Components[i].ParameterValues)
		assert.Equal(comp.Scopes, app.Spec.Components[i].Scopes)
		assert.Equal(comp.Traits, app.Spec.Components[i].Traits)
	}

}

// getSampleMCAppConfig creates and returns a sample MultiClusterApplicationConfiguration used in tests
func getSampleMCAppConfig() (clustersv1alpha1.MultiClusterApplicationConfiguration, error) {
	mcAppConfig := clustersv1alpha1.MultiClusterApplicationConfiguration{}
	sampleMCAppConfigFile, err := filepath.Abs("testdata/hello-multiclusterappconfig.yaml")
	if err != nil {
		return mcAppConfig, err
	}

	rawMCAppConfig, err := clusterstest.ReadYaml2Json(sampleMCAppConfigFile)
	if err != nil {
		return mcAppConfig, err
	}

	err = json.Unmarshal(rawMCAppConfig, &mcAppConfig)

	return mcAppConfig, err
}

func getExistingOAMAppConfig() (v1alpha2.ApplicationConfiguration, error) {
	oamAppConfig := v1alpha2.ApplicationConfiguration{}
	existingAppConfigFile, err := filepath.Abs("testdata/hello-oam-appconfig-existing.yaml")
	if err != nil {
		return oamAppConfig, err
	}
	rawMcAppConfig, err := clusterstest.ReadYaml2Json(existingAppConfigFile)
	if err != nil {
		return oamAppConfig, err
	}

	err = json.Unmarshal(rawMcAppConfig, &oamAppConfig)
	return oamAppConfig, err
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
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}
