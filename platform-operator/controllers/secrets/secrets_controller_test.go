// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func expectLocalCABundleIsCreated(t *testing.T, mock *mocks.MockClient) {
	asserts := assert.New(t)

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
	scheme := newScheme()
	reconciler := VerrazzanoSecretsReconciler{
		Client: c,
		Scheme: scheme}
	return reconciler
}
