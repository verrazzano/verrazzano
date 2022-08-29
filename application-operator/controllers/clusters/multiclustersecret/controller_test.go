// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustersecret

import (
	"context"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

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

const namespace = "unit-mcsecret-namespace"
const crName = "unit-mcsecret"

// TestReconcilerSetupWithManager test the creation of the Reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
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
	doExpectGetMultiClusterSecret(cli, mcSecretSample, false)

	// expect a call to fetch the managed cluster registration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing corev1.Secret and return not found error, to test create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, crName))

	// expect a call to create the K8S secret
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			assertSecretValid(assert, s, secretData)
			return nil
		})

	// expect a call to update the resource with a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			assert.True(len(secret.ObjectMeta.Finalizers) == 1, "Wrong number of finalizers")
			assert.Equal(finalizerName, secret.ObjectMeta.Finalizers[0], "wrong finalizer")
			return nil
		})

	// expect a call to update the status of the multicluster secret
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

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
	doExpectGetMultiClusterSecret(cli, mcSecretSample, true)

	// expect a call to fetch the managed cluster registration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch underlying secret, and return an existing secret
	doExpectGetSecretExists(cli, mcSecretSample.ObjectMeta, existingSecretData)

	// expect a call to update the K8S secret with the new secret data
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.UpdateOption) error {
			assertSecretValid(assert, s, newSecretData)
			return nil
		})

	// expect a call to update the status of the multicluster secret
	doExpectStatusUpdateSucceeded(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

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
	doExpectGetMultiClusterSecret(cli, mcSecretSample, false)

	// expect a call to fetch the managed cluster registration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing corev1.Secret and return not found error, to simulate create case
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Secret"}, crName))

	// expect a call to create the K8S secret and fail the call
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.CreateOption) error {
			return errors.NewBadRequest("will not create it")
		})

	// expect that the status of MultiClusterSecret is updated to failed because we
	// failed the underlying secret's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
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
	doExpectGetMultiClusterSecret(cli, mcSecretSample, true)

	// expect a call to fetch the managed cluster registration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to fetch existing corev1.Secret (simulate update case)
	doExpectGetSecretExists(cli, mcSecretSample.ObjectMeta, existingSecretData)

	// expect a call to update the K8S secret and fail the call
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, s *v1.Secret, opts ...client.UpdateOption) error {
			return errors.NewBadRequest("will not update it")
		})

	// expect that the status of MultiClusterSecret is updated to failed because we
	// failed the underlying secret's creation
	doExpectStatusUpdateFailed(cli, mockStatusWriter, assert)

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.Equal(true, result.Requeue)
}

// TestReconcilePlacementInDifferentCluster tests the path of reconciling a MultiClusterSecret which
// is placed on a cluster other than the current cluster. We expect this MultiClusterSecret to
// be ignored, and no K8S secret to be created
// GIVEN a MultiClusterSecret resource is created with a placement in different cluster
// WHEN the controller Reconcile function is called
// THEN expect that no K8S Secret is created
func TestReconcilePlacementInDifferentCluster(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	statusWriter := mocks.NewMockStatusWriter(mocker)

	secretData := map[string][]byte{"username": []byte("aaaaa")}

	mcSecretSample := getSampleMCSecret(namespace, crName, secretData)

	mcSecretSample.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// expect a call to fetch the MultiClusterSecret
	doExpectGetMultiClusterSecret(cli, mcSecretSample, true)

	// expect a call to fetch the MCRegistration secret
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// The effective state of the object will get updated even if it is note locally placed,
	// since it would have changed
	clusterstest.DoExpectUpdateState(t, cli, statusWriter, &mcSecretSample, clustersv1alpha1.Pending)

	clusterstest.ExpectDeleteAssociatedResource(cli, &v1alpha2.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcSecretSample.Name,
			Namespace: mcSecretSample.Namespace,
		},
	}, types.NamespacedName{
		Namespace: mcSecretSample.Namespace,
		Name:      mcSecretSample.Name,
	})

	// expect a call to update the resource with no finalizers
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			assert.True(len(mcSecret.Finalizers) == 0, "Wrong number of finalizers")
			return nil
		})

	// Expect no further action

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileResourceNotFound tests the path of reconciling a
// MultiClusterSecret resource which is non-existent when reconcile is called,
// possibly because it has been deleted.
// GIVEN a MultiClusterSecret resource has been deleted
// WHEN the controller Reconcile function is called
// THEN expect that no action is taken
func TestReconcileResourceNotFound(t *testing.T) {
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// expect a call to fetch the MultiClusterLoggingScope
	// and return a not found error
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: clustersv1alpha1.SchemeGroupVersion.Group, Resource: clustersv1alpha1.MultiClusterSecretResource}, crName))

	// expect no further action to be taken

	// create a request and reconcile it
	request := clusterstest.NewRequest(namespace, crName)
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

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
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the multicluster secret
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterSecret{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcSecret.Status,
				clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, v1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of MultiClusterSecret to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the multicluster secret
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the multicluster secret
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.MultiClusterSecret{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, mcSecret.Status,
				clustersv1alpha1.Succeeded, clustersv1alpha1.DeployComplete, v1.ConditionTrue)
			return nil
		})
}

// doExpectGetMultiClusterSecret adds an expectation to the given MockClient to expect a Get
// call for a MultiClusterSecret, and populate the multi cluster secret with given data
func doExpectGetMultiClusterSecret(cli *mocks.MockClient, mcSecretSample clustersv1alpha1.MultiClusterSecret, addFinalizer bool) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: crName}, gomock.AssignableToTypeOf(&mcSecretSample)).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			mcSecret.ObjectMeta = mcSecretSample.ObjectMeta
			mcSecret.TypeMeta = mcSecretSample.TypeMeta
			mcSecret.Spec = mcSecretSample.Spec
			if addFinalizer {
				mcSecret.Finalizers = append(mcSecret.Finalizers, finalizerName)
			}
			return nil
		})
}

// assertSecretValid asserts that the metadata and content of the created/updated K8S secret
// are valid
func assertSecretValid(assert *asserts.Assertions, s *v1.Secret, secretData map[string][]byte) {
	assert.Equal(v1.SecretTypeOpaque, s.Type)
	assert.Equal(namespace, s.ObjectMeta.Namespace)
	assert.Equal(crName, s.ObjectMeta.Name)
	assert.Equal(secretData, s.Data)
	// assert that the secret is labeled verrazzano-managed=true since it was created by Verrazzano
	assert.NotNil(s.Labels)
	assert.Equal(constants.LabelVerrazzanoManagedDefault, s.Labels[vzconst.VerrazzanoManagedLabelKey])
}

// getSampleMCSecret creates and returns a sample MultiClusterSecret used in tests
func getSampleMCSecret(ns string, name string, secretData map[string][]byte) clustersv1alpha1.MultiClusterSecret {
	var mcSecret clustersv1alpha1.MultiClusterSecret
	mcSecret.Spec.Template = clustersv1alpha1.SecretTemplate{Type: v1.SecretTypeOpaque, Data: secretData}
	mcSecret.ObjectMeta.Namespace = namespace
	mcSecret.ObjectMeta.Name = crName
	mcSecret.APIVersion = clustersv1alpha1.SchemeGroupVersion.String()
	mcSecret.Kind = "MultiClusterSecret"
	mcSecret.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: clusterstest.UnitTestClusterName}}
	return mcSecret
}

// newSecretReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newSecretReconciler(c client.Client) Reconciler {
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
	reconciler := newSecretReconciler(cli)
	result, err := reconciler.Reconcile(nil, request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}
