// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
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
	"testing"
)

const namespace = "unit-mcsecret-namespace"
const crName = "unit-mcsecret"

// TestReconcilerSetupWithManager test the creation of the MultiClusterSecretReconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler MultiClusterSecretReconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	reconciler = MultiClusterSecretReconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileCreateSecret tests the basic happy path of reconciling a MultiClusterSecret. We
// expect to write out a K8S secret
// GIVEN a MultiClusterSecret resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Secret to be created
func TestReconcileCreateSecret(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	secretData := map[string][]byte{"username": []byte("aaaaa")}

	mcSecretSample := getSampleMCSecret(namespace, crName, secretData)

	// expect a call to fetch the MultiClusterSecret
	doExpectGetMultiClusterSecret(cli, mcSecretSample)

	// expect a call to fetch existing corev1.Secret and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Secret"}, crName))

	// expect a call to create the K8S secret
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			assertSecretValid(assert, s, secretData)
			return nil
		})

	// expect a call to update the status of the multicluster secret
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateSecret tests the path of reconciling a MultiClusterSecret when the underlying
// secret already exists i.e. update
// expect to update a K8S secret
// GIVEN a MultiClusterSecret resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Secret to be updated
func TestReconcileUpdateSecret(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	newSecretData := map[string][]byte{"username": []byte("aaaaa")}
	existingSecretData := map[string][]byte{"username": []byte("existing")}

	mcSecretSample := getSampleMCSecret(namespace, crName, newSecretData)

	// expect a call to fetch the MultiClusterSecret
	doExpectGetMultiClusterSecret(cli, mcSecretSample)

	// expect a call to fetch underlying secret, and return an existing secret
	doExpectGetSecretExists(cli, mcSecretSample.ObjectMeta, existingSecretData)

	// expect a call to update the K8S secret with the new secret data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			assertSecretValid(assert, s, newSecretData)
			return nil
		})

	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&mcSecretSample)).
		Return(nil)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateSecretFailed tests the path of reconciling a MultiClusterSecret
// when the underlying secret does not exist and fails to be created due to some error condition
// GIVEN a MultiClusterSecret resource is created
// WHEN the controller Reconcile function is called and create underlying secret fails
// THEN expect the status of the MultiClusterSecret to be updated with failure information
func TestReconcileCreateSecretFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	secretData := map[string][]byte{"username": []byte("aaaaa")}

	mcSecretSample := getSampleMCSecret(namespace, crName, secretData)

	// expect a call to fetch the MultiClusterSecret
	doExpectGetMultiClusterSecret(cli, mcSecretSample)

	// expect a call to fetch existing corev1.Secret and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Secret"}, crName))

	// expect a call to create the K8S secret and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterSecret is updated to failed because we
	// failed the underlying secret's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileUpdateSecretFailed(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	secretData := map[string][]byte{"username": []byte("aaaaa")}
	existingSecretData := map[string][]byte{"username": []byte("existing secret data")}

	mcSecretSample := getSampleMCSecret(namespace, crName, secretData)

	// expect a call to fetch the MultiClusterSecret
	doExpectGetMultiClusterSecret(cli, mcSecretSample)

	// expect a call to fetch existing corev1.Secret (simulate update case)
	doExpectGetSecretExists(cli, mcSecretSample.ObjectMeta, existingSecretData)

	// expect a call to update the K8S secret and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterSecret is updated to failed because we
	// failed the underlying secret's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := newRequest(namespace, crName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// doExpectGetSecretExists expects a call to get a corev1.Secret, and return an "existing" secret
func doExpectGetSecretExists(cli *mocks.MockClient, metadata metav1.ObjectMeta, existingSecretData map[string][]byte) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Data = existingSecretData
			secret.ObjectMeta = metadata
			return nil
		})
}

// doExpectStatusUpdateFailed expects a call to update status of MultiClusterSecret to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the multicluster secret
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterSecret{})).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			assertMultiClusterSecretStatus(assert, mcSecret, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterSecret to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the multicluster secret
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterSecret{})).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			assertMultiClusterSecretStatus(assert, mcSecret, clustersv1alpha1.Ready, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterSecret adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterSecret, and populate the multi cluster secret with given data
func doExpectGetMultiClusterSecret(cli *mocks.MockClient, mcSecretSample clustersv1alpha1.MultiClusterSecret) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcSecretSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			mcSecret.ObjectMeta = mcSecretSample.ObjectMeta
			mcSecret.TypeMeta = mcSecretSample.TypeMeta
			mcSecret.Spec = mcSecretSample.Spec
			return nil
		})
}

// assertMultiClusterSecretStatus asserts that the status and conditions on the MultiClusterSecret
// are as expected
func assertMultiClusterSecretStatus(assert *asserts.Assertions, mcSecret *clustersv1alpha1.MultiClusterSecret, state clustersv1alpha1.StateType, condition clustersv1alpha1.ConditionType, conditionStatus v1.ConditionStatus) {
	assert.Equal(state, mcSecret.Status.State)
	assert.Equal(1, len(mcSecret.Status.Conditions))
	assert.Equal(conditionStatus, mcSecret.Status.Conditions[0].Status)
	assert.Equal(condition, mcSecret.Status.Conditions[0].Type)
}

// assertSecretValid asserts that the metadata and content of the created/updated K8S secret
// are valid
func assertSecretValid(assert *asserts.Assertions, s *v1.Secret, secretData map[string][]byte) {
	assert.Equal(v1.SecretTypeOpaque, s.Type)
	assert.Equal(namespace, s.ObjectMeta.Namespace)
	assert.Equal(crName, s.ObjectMeta.Name)
	assert.Equal(secretData, s.Data)
	assert.Equal(1, len(s.OwnerReferences))
	assert.Equal("MultiClusterSecret", s.OwnerReferences[0].Kind)
	assert.Equal(clustersv1alpha1.GroupVersion.String(), s.OwnerReferences[0].APIVersion)
	assert.Equal(crName, s.OwnerReferences[0].Name)
}

// getSampleMCSecret creates and returns a sample MultiClusterSecret used in tests
func getSampleMCSecret(ns string, name string, secretData map[string][]byte) clustersv1alpha1.MultiClusterSecret {
	var mcSecret clustersv1alpha1.MultiClusterSecret
	mcSecret.Spec.Template = clustersv1alpha1.SecretTemplate{Type: v1.SecretTypeOpaque, Data: secretData}
	mcSecret.ObjectMeta.Namespace = namespace
	mcSecret.ObjectMeta.Name = crName
	mcSecret.APIVersion = clustersv1alpha1.GroupVersion.String()
	mcSecret.Kind = "MultiClusterSecret"
	mcSecret.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: "myCluster"}}
	return mcSecret
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) MultiClusterSecretReconciler {
	return MultiClusterSecretReconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
	}
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}
