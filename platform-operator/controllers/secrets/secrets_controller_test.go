// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var mcNamespace = types.NamespacedName{Name: constants.VerrazzanoMultiClusterNamespace}
var vzTLSSecret = types.NamespacedName{Name: "verrazzano-tls", Namespace: "verrazzano-system"}
var vzLocalCaBundleSecret = types.NamespacedName{Name: "verrazzano-local-ca-bundle", Namespace: "verrazzano-mc"}
var unwatchedSecret = types.NamespacedName{Name: "any-secret", Namespace: "any-namespace"}

// TestCreateLocalCABundle tests the Reconcile method for the following use case
// GIVEN a request to reconcile the verrazzano-tls secret
// WHEN the local-ca-bundle secret doesn't exist
// THEN the local-ca-bundle secret is updated
func TestCreateLocalCABundle(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	expectLocalCABundleIsCreated(t, mock)

	// Create and make the request
	request := newRequest(vzTLSSecret.Namespace, vzTLSSecret.Name)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.NotNil(result)
}

// TestIgnoresOtherSecrets tests the Reconcile method for the following use case
// GIVEN a request to reconcile a secret other than verrazzano-tls
// WHEN any conditions
// THEN the request is ignored
func TestIgnoresOtherSecrets(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	expectNothingForWrongSecret(t, mock)

	// Create and make the request
	request := newRequest(unwatchedSecret.Namespace, unwatchedSecret.Name)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.NotNil(result)
}

// TestMultiClusterNamespaceDoesNotExist tests the Reconcile method for the following use case
// GIVEN a request to reconcile the verrazzano-tls secret
// WHEN the verrazzano-mc namespace does not exist
// THEN a requeue request is returned with no error
func TestMultiClusterNamespaceDoesNotExist(t *testing.T) {
	runNamespaceErrorTest(t, errors.NewNotFound(corev1.Resource("Namespace"), constants.VerrazzanoMultiClusterNamespace))
}

// TestMultiClusterNamespaceUnexpectedErr tests the Reconcile method for the following use case
// GIVEN a request to reconcile the verrazzano-tls secret
// WHEN an unexpected error occurs checking the verrazzano-mc namespace existence
// THEN a requeue request is returned with no error
func TestMultiClusterNamespaceUnexpectedErr(t *testing.T) {
	runNamespaceErrorTest(t, fmt.Errorf("unexpected error checking namespace"))
}

// TestSecretReconciler tests Reconciler method for secrets controller.
func TestSecretReconciler(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &testSecret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request)

	asserts.NoError(err0)
	asserts.Empty(res0)

}

func runNamespaceErrorTest(t *testing.T, expectedErr error) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect  a call to get the verrazzano-mc namespace
	mock.EXPECT().
		Get(gomock.Any(), mcNamespace, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return expectedErr
		})

	// Expect a call to get the verrazzano-tls secret
	mock.EXPECT().
		Get(gomock.Any(), vzTLSSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = vzTLSSecret.Name
			secret.Namespace = vzTLSSecret.Namespace
			return nil
		}).AnyTimes()

	// Create and make the request
	request := newRequest(vzTLSSecret.Namespace, vzTLSSecret.Name)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.NotNil(result)
	asserts.NotEqual(ctrl.Result{}, result)
}

// TestSecretRequeue tests that we requeue if Component Status hasn't been
// initialized by Verrazzano
func TestSecretRequeue(t *testing.T) {
	asserts := assert.New(t)
	vz := testVZ
	vz.Status.Components = nil
	asserts.Nil(vz.Status.Components)
	cli := fake.NewClientBuilder().WithObjects(&vz, &testSecret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.Error(err0)
	asserts.Equal(true, res0.Requeue)
}

// TestSecretCall tests that the call to get the ConfigMap is placed
func TestSecretCall(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	expectGetSecretExists(mock, &testSecret, testNS, testSecretName)

	request := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.reconcileHelmOverrideSecret(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestOtherNS tests that the API call to get the Secret is not made
// if the request namespace does not match Verrazzano Namespace
func TestOtherNS(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Do not expect a call to get the Secret if it's a different namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).MaxTimes(0)

	request := newRequest("test0", "test1")
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.reconcileHelmOverrideSecret(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

}

// mock client request to get the secret
func expectGetSecretExists(mock *mocks.MockClient, SecretToUse *corev1.Secret, namespace string, name string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			return nil
		})
}

func expectLocalCABundleIsCreated(t *testing.T, mock *mocks.MockClient) {
	asserts := assert.New(t)

	// Expect  a call to get the verrazzano-mc namespace
	mock.EXPECT().
		Get(gomock.Any(), mcNamespace, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return nil
		}).AnyTimes()

	// Expect a call to get the verrazzano-tls secret
	mock.EXPECT().
		Get(gomock.Any(), vzTLSSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = vzTLSSecret.Name
			secret.Namespace = vzTLSSecret.Namespace
			return nil
		}).AnyTimes()

	// Expect a call to get the verrazzano-tls secret
	mock.EXPECT().
		Get(gomock.Any(), vzLocalCaBundleSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret2 *corev1.Secret) error {
			secret2.Name = vzLocalCaBundleSecret.Name
			secret2.Namespace = vzLocalCaBundleSecret.Namespace
			return nil
		}).AnyTimes()

	// Expect a call to update the verrazzano-local-ca-bundle finalizer
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			asserts.Equal(vzLocalCaBundleSecret.Name, secret.Name, "wrong secret name")
			asserts.Equal(vzLocalCaBundleSecret.Namespace, secret.Namespace, "wrong secret namespace")
			return nil
		})
}

func expectNothingForWrongSecret(t *testing.T, mock *mocks.MockClient) {

	mock.EXPECT().
		List(gomock.Any(), &vzapi.VerrazzanoList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzanoList *vzapi.VerrazzanoList, options ...client.ListOption) error {
			return nil
		})
	// Expect no calls to get a secret
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).MaxTimes(0)

	// Expect no calls to get update
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
}

// newSecretsReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newSecretsReconciler(c client.Client) VerrazzanoSecretsReconciler {
	vzLog := vzlog.DefaultLogger()
	scheme := newScheme()
	reconciler := VerrazzanoSecretsReconciler{
		Client: c,
		Scheme: scheme,
		log:    vzLog}
	return reconciler
}
