// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
var vzTLSSecret = types.NamespacedName{Name: constants.VerrazzanoIngressSecret, Namespace: constants.VerrazzanoSystemNamespace}
var additionalTLSSecret = types.NamespacedName{Name: constants2.AdditionalTLS, Namespace: constants2.RancherSystemNamespace}
var vzLocalCaBundleSecret = types.NamespacedName{Name: "verrazzano-local-ca-bundle", Namespace: constants.VerrazzanoMultiClusterNamespace}
var unwatchedSecret = types.NamespacedName{Name: "any-secret", Namespace: "any-namespace"}

const addnlTLSData = "YWRkaXRpb25hbCB0bHMgc2VjcmV0" // "additional tls secret"

// TestCreateLocalCABundle tests the Reconcile method for the following use cases
// GIVEN a request to reconcile the verrazzano-tls secret OR the tls-additional-ca secret
// WHEN the local-ca-bundle secret doesn't exist
// THEN the local-ca-bundle secret is updated
func TestCreateLocalCABundle(t *testing.T) {
	tests := []struct {
		secretName     string
		secretNS       string
		secretKey      string
		secretData     string
		addnlTLSExists bool
	}{
		{
			secretName:     vzTLSSecret.Name,
			secretNS:       vzTLSSecret.Namespace,
			secretKey:      "ca.crt",
			secretData:     "dnogdGxzIHNlY3JldA==", // "vz tls secret",
			addnlTLSExists: false,
		},
		{
			secretName:     vzTLSSecret.Name,
			secretNS:       vzTLSSecret.Namespace,
			secretKey:      "ca.crt",
			secretData:     "dnogdGxzIHNlY3JldA==", // "vz tls secret",
			addnlTLSExists: true,
		},
		{
			secretName:     additionalTLSSecret.Name,
			secretNS:       additionalTLSSecret.Namespace,
			secretKey:      constants2.AdditionalTLSCAKey,
			secretData:     addnlTLSData,
			addnlTLSExists: true,
		},
	}
	for _, tt := range tests {
		asserts := assert.New(t)
		mocker := gomock.NewController(t)
		mock := mocks.NewMockClient(mocker)

		isAddnlTLSSecret := (tt.secretName == additionalTLSSecret.Name)

		if !isAddnlTLSSecret {
			// When reconciling secrets other than additionalTLS secret, expect a call to check if
			// additional TLS secret exists. Expect the local secret to be updated ONLY if additional
			// TLS doesn't exist
			expectGetAdditionalTLS(t, mock, tt.addnlTLSExists, "")
		}

		// only expect reconcile to happen if we are reconciling the additional TLS secret, OR
		// we are reconciling another secret but the additional TLS secret does NOT exist
		if isAddnlTLSSecret || !tt.addnlTLSExists {
			expectGetCalls(t, mock, tt.secretNS, tt.secretName, tt.secretKey, tt.secretData)
			expectUpdateLocalSecret(t, mock, tt.secretData)
		}

		// Create and make the request
		request := newRequest(tt.secretNS, tt.secretName)
		reconciler := newSecretsReconciler(mock)
		result, err := reconciler.Reconcile(context.TODO(), request)

		// Validate the results
		mocker.Finish()
		asserts.NoError(err)
		asserts.NotNil(result)
	}
}

// TestIgnoresOtherSecrets tests the Reconcile method for the following use case
// GIVEN a request to reconcile a secret other than verrazzano TLS secret or additional TLS secret
// WHEN any conditions
// THEN the request is ignored
func TestIgnoresOtherSecrets(t *testing.T) {
	tests := []struct {
		secretName string
		secretNS   string
	}{
		// VZ TLS secret name in wrong NS
		{
			secretName: vzTLSSecret.Name,
			secretNS:   additionalTLSSecret.Namespace,
		},
		// Additional TLS secret name in wrong NS
		{
			secretName: additionalTLSSecret.Name,
			secretNS:   vzTLSSecret.Namespace,
		},
		// A totally different secret name and NS
		{
			secretName: unwatchedSecret.Name,
			secretNS:   unwatchedSecret.Namespace,
		},
	}
	for _, tt := range tests {
		asserts := assert.New(t)
		mocker := gomock.NewController(t)
		mock := mocks.NewMockClient(mocker)

		expectNothingForWrongSecret(t, mock)

		// Create and make the request
		request := newRequest(tt.secretNS, tt.secretName)
		reconciler := newSecretsReconciler(mock)
		result, err := reconciler.Reconcile(context.TODO(), request)

		// Validate the results
		mocker.Finish()
		asserts.NoError(err)
		asserts.NotNil(result)
	}
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
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.KubeFinalizer)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request)

	asserts.NoError(err0)
	asserts.Empty(res0)

	vz := vzapi.Verrazzano{}
	err := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, &vz)
	asserts.NoError(err)
	asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)

}

// TestSecretRequeue tests that we requeue if Component Status hasn't been
// initialized by Verrazzano
func TestSecretRequeue(t *testing.T) {
	asserts := assert.New(t)
	vz := testVZ
	vz.Status.Components = nil
	asserts.Nil(vz.Status.Components)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.KubeFinalizer)
	cli := fake.NewClientBuilder().WithObjects(&vz, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.Error(err0)
	asserts.Equal(true, res0.Requeue)
}

// TestFinalizer tests that a finalizer is added to Secret that's missing
// and there is a requeue and the finalizer is added
func TestFinalizer(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &testSecret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)

	secret := corev1.Secret{}
	err := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testSecretName}, &secret)
	asserts.NoError(err)
	asserts.NotZero(len(secret.Finalizers))
}

// TestOtherFinalizers tests that if more than one finalizers are present
// the we requeue
func TestOtherFinalizers(t *testing.T) {
	asserts := assert.New(t)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.KubeFinalizer, "test")
	secret.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)
}

// TestDeletion tests that if the object is scheduled for deletion with
// only our finalizer, then the finalizer is removed and Verrazzano resource is updated
func TestDeletion(t *testing.T) {
	asserts := assert.New(t)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.KubeFinalizer)
	secret.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(false, res0.Requeue)

	sec1 := &corev1.Secret{}
	err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testSecretName}, sec1)
	asserts.True(errors.IsNotFound(err1))

	vz := &vzapi.Verrazzano{}
	err2 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
	asserts.NoError(err2)
	asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
}

// TestSecretCall tests that the call to get the Secret is placed
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

	// Expect a call to get the verrazzano-tls secret
	mock.EXPECT().
		Get(gomock.Any(), additionalTLSSecret, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants2.RancherSystemNamespace, Resource: "Secret"}, additionalTLSSecret.Name)).
		MinTimes(1)

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

func expectGetAdditionalTLS(t *testing.T, mock *mocks.MockClient, exists bool, secretData string) {
	// Expect a call to get the additional-tls secret (to check if it exists), and return
	// one if exists == true, otherwise return not found
	if exists {
		mock.EXPECT().
			Get(gomock.Any(), additionalTLSSecret, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
				secret.Name = additionalTLSSecret.Name
				secret.Namespace = additionalTLSSecret.Namespace
				secret.Data = map[string][]byte{constants2.AdditionalTLSCAKey: []byte(secretData)}
				return nil
			}).MinTimes(1)
	} else {
		mock.EXPECT().
			Get(gomock.Any(), additionalTLSSecret, gomock.Not(gomock.Nil())).
			Return(errors.NewNotFound(schema.GroupResource{Group: constants2.RancherSystemNamespace, Resource: "Secret"}, additionalTLSSecret.Name)).
			MinTimes(1)
	}
}

// mock client request to get the secret
func expectGetSecretExists(mock *mocks.MockClient, SecretToUse *corev1.Secret, namespace string, name string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			return nil
		})
}

func expectGetCalls(t *testing.T, mock *mocks.MockClient, secretNS string, secretName string, secretKey string, secretData string) {
	// Expect  a call to get the verrazzano-mc namespace
	mock.EXPECT().
		Get(gomock.Any(), mcNamespace, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return nil
		}).AnyTimes()

	// Expect a call to get the specified tls secret
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: secretName, Namespace: secretNS}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret) error {
			secret.Name = secretName
			secret.Namespace = secretNS
			secret.Data = map[string][]byte{secretKey: []byte(secretData)}
			return nil
		}).MinTimes(1)

	// Expect a call to get the local ca bundle secret
	mock.EXPECT().
		Get(gomock.Any(), vzLocalCaBundleSecret, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret2 *corev1.Secret) error {
			secret2.Name = vzLocalCaBundleSecret.Name
			secret2.Namespace = vzLocalCaBundleSecret.Namespace
			return nil
		}).MinTimes(1)
}

func expectUpdateLocalSecret(t *testing.T, mock *mocks.MockClient, expectedSecretData string) {
	asserts := assert.New(t)
	// Expect a call to update the verrazzano-local-ca-bundle
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.UpdateOption) error {
			asserts.Equal(vzLocalCaBundleSecret.Name, secret.Name, "wrong secret name")
			asserts.Equal(vzLocalCaBundleSecret.Namespace, secret.Namespace, "wrong secret namespace")
			asserts.Equal([]byte(expectedSecretData), secret.Data["ca-bundle"], "wrong secret ca-bundle")
			return nil
		}).MinTimes(1)
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
	scheme := newScheme()
	reconciler := VerrazzanoSecretsReconciler{
		Client: c,
		Scheme: scheme}
	return reconciler
}
