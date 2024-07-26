// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	certv1fake "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	certv1client "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcommonfake "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common/fake"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var vzTLSSecret = types.NamespacedName{Name: constants.VerrazzanoIngressSecret, Namespace: constants.VerrazzanoSystemNamespace}
var vzPrivateCABundleSecret = types.NamespacedName{Name: constants2.PrivateCABundle, Namespace: constants.VerrazzanoSystemNamespace}
var additionalTLSSecret = types.NamespacedName{Name: "tls-ca-additional", Namespace: constants2.RancherSystemNamespace}
var unwatchedSecret = types.NamespacedName{Name: "any-secret", Namespace: "any-namespace"}

// TestReconcileConfiguredCASecret tests the Reconcile method
// GIVEN a request to reconcile the secret configured in ClusterIssuer
// WHEN the secret has changed
// THEN verify all certificates managed by ClusterIssuer are rotated
// THEN verify the verrazzano-system/verrazzano-tls-ca secret is updated with the changes
// THEN verify the cattle-system/tls-ca secret is updated with the changes
// THEN verify the verrazzano-mc/verrazzano-local-ca-bundle secret is updated with the changes
func TestReconcileConfiguredCASecret(t *testing.T) {
	const caCertCommonName = "verrazzano-root-ca"
	asserts := assert.New(t)
	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	scheme := newScheme()
	vz := newVZ()

	// Create a CA certificate
	commonName := caCertCommonName + "-a23asdfa"
	caIssuerCert := cmcommonfake.CreateFakeCertificate(commonName)
	caSecret, caCert, err := newCertificateWithSecret("verrazzano-selfsigned-issuer", commonName, "verrazzano-ca-certificate", constants2.CertManagerNamespace, nil)
	asserts.NoError(err)

	// Create a leaf certificate signed by the CA
	leaf1Secret, leaf1Cert, err := newCertificateWithSecret("verrazzano-cluster-issuer", "common-name", "tls-rancher-ingress", constants2.RancherSystemNamespace, caIssuerCert)
	assert.NoError(t, err)
	leaf1Secret.Data[constants2.CACertKey] = caSecret.Data[corev1.TLSCertKey]

	// Create the verrazzano-tls-ca secret
	v8oTLSCASecret := newCertSecret(constants2.PrivateCABundle, constants2.VerrazzanoSystemNamespace, constants2.CABundleKey, caSecret.Data[corev1.TLSCertKey])

	// Create the Rancher tls-ca secret
	cattleTLSSecret := newCertSecret(constants2.RancherTLSCA, constants2.RancherSystemNamespace, constants2.RancherTLSCAKey, caSecret.Data[corev1.TLSCertKey])

	// Create the Rancher deployment
	cattleDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants2.RancherSystemNamespace,
			Name:      rancherDeploymentName,
		},
	}

	// Create the multi-cluster namespace
	multiClusterNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants2.VerrazzanoMultiClusterNamespace,
		},
	}

	// Create the multi-cluster verrazzano-local-ca-bundle secret
	mcSecret := newCertSecret(constants.VerrazzanoLocalCABundleSecret, constants.VerrazzanoMultiClusterNamespace, mcCABundleKey, caSecret.Data[corev1.TLSCertKey])

	// Simulate rotate of the CA cert
	fakeIssuerCertBytes, err := cmcommonfake.CreateFakeCertBytes(commonName+"foo", nil)
	assert.NoError(t, err)
	caSecret.Data[corev1.TLSCertKey] = fakeIssuerCertBytes

	// Fake ControllerRuntime client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vz, caSecret, caCert, leaf1Secret, leaf1Cert,
		v8oTLSCASecret, cattleTLSSecret, cattleDeployment, multiClusterNamespace, mcSecret).Build()
	r := newSecretsReconciler(fakeClient)

	// Fake Go client for the CertManager clientSet
	cmClient := certv1fake.NewSimpleClientset(caCert, leaf1Cert)
	defer issuer.ResetCMClientFunc()
	issuer.SetCMClientFunc(func() (certv1client.CertmanagerV1Interface, error) {
		return cmClient.CertmanagerV1(), nil
	})

	// First reconcile the change to the ClusterIssuer secret
	request := newRequest(caSecret.Namespace, caSecret.Name)
	result, err := r.Reconcile(context.TODO(), request)
	asserts.NoError(err)
	asserts.NotNil(result)

	// Next reconcile the change to the verrazzano-tls-ca secret
	request = newRequest(v8oTLSCASecret.Namespace, v8oTLSCASecret.Name)
	result, err = r.Reconcile(context.TODO(), request)
	asserts.NoError(err)
	asserts.NotNil(result)

	// Confirm the expected certificates were marked to be rotated
	updatedCert, err := cmClient.CertmanagerV1().Certificates(leaf1Cert.Namespace).Get(context.TODO(), leaf1Cert.Name, metav1.GetOptions{})
	asserts.NoError(err)
	asserts.True(cmutil.CertificateHasCondition(updatedCert, certv1.CertificateCondition{
		Type:   certv1.CertificateConditionIssuing,
		Status: cmmeta.ConditionTrue,
	}))

	// Confirm the verrazzano-tls-ca secret got updated
	secret := &corev1.Secret{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: v8oTLSCASecret.Namespace, Name: v8oTLSCASecret.Name}, secret)
	asserts.NoError(err)
	asserts.Equal(caSecret.Data[corev1.TLSCertKey], secret.Data[constants2.CABundleKey])

	// Confirm the Rancher tls-ca secret got updated
	secret = &corev1.Secret{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: cattleTLSSecret.Namespace, Name: cattleTLSSecret.Name}, secret)
	asserts.NoError(err)
	asserts.Equal(caSecret.Data[corev1.TLSCertKey], secret.Data[constants2.RancherTLSCAKey])

	// Confirm the Rancher deployment was annotated to restart
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: cattleDeployment.Namespace, Name: cattleDeployment.Name}, deployment)
	asserts.NoError(err)
	annotations := deployment.Spec.Template.ObjectMeta.Annotations
	asserts.NotNil(annotations)
	asserts.NotEmpty(annotations[constants2.VerrazzanoRestartAnnotation])

	// Confirm the multi-cluster verrazzano-local-ca-bundle secret got updated
	secret = &corev1.Secret{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: mcSecret.Namespace, Name: mcSecret.Name}, secret)
	asserts.NoError(err)
	asserts.Equal(caSecret.Data[corev1.TLSCertKey], secret.Data[mcCABundleKey])
}

// TestIgnoresOtherSecrets tests the Reconcile method for the following use case
// GIVEN a request to reconcile the additional TLS secret or a secret other than verrazzano TLS secret
// WHEN any conditions
// THEN the request is ignored
func TestIgnoresOtherSecrets(t *testing.T) {
	tests := []struct {
		secretName string
		secretNS   string
	}{
		// Additional TLS secret no longer watched
		{
			secretName: additionalTLSSecret.Name,
			secretNS:   additionalTLSSecret.Namespace,
		},
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

		expectNothingForWrongSecret(mock)

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

// TestSecretReconciler tests the Reconciler method for the following use case
// GIVEN a request to reconcile a Secret
// WHEN the Secret is referenced in the Verrazzano CR under a component and is also present the CR namespace
// THEN the ReconcilingGeneration of the target component is set to 1
func TestSecretReconciler(t *testing.T) {
	asserts := assert.New(t)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.OverridesFinalizer)
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

// TestSecretRequeue tests the Reconcile method for the following use case
// GIVEN a request to reconcile a Secret that qualifies as an override
// WHEN the status of the Verrazzano CR is found without the Component Status details
// THEN a requeue request is returned with an error
func TestSecretRequeue(t *testing.T) {
	asserts := assert.New(t)
	vz := testVZ
	vz.Status.Components = nil
	asserts.Nil(vz.Status.Components)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.OverridesFinalizer)
	cli := fake.NewClientBuilder().WithObjects(&vz, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.Error(err0)
	asserts.Contains(err0.Error(), "Components not initialized")
	asserts.Equal(true, res0.Requeue)
}

// TestAddFinalizer tests the Reconciler for the following use case
// GIVEN a request to reconcile a Secret that qualifies as an override
// WHEN the Secret is found without the overrides finalizer
// THEN the overrides finalizer is added and we requeue without an error
func TestAddFinalizer(t *testing.T) {
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
	asserts.True(controllerutil.ContainsFinalizer(&secret, constants.OverridesFinalizer))
}

// TestOtherFinalizers tests the Reconcile loop for the following use case
// GIVEN a request to reconcile a Secret that qualifies as an override resource and is scheduled for deletion
// WHEN the Secret is found with finalizers but the override finalizer is missing
// THEN without updating the Verrazzano CR a requeue request is returned without an error
func TestOtherFinalizers(t *testing.T) {
	asserts := assert.New(t)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, "test")
	secret.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cli := fake.NewClientBuilder().WithObjects(&testVZ, &secret).WithScheme(newScheme()).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	request0 := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(cli)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)

	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)

	vz := &vzapi.Verrazzano{}
	err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
	asserts.NoError(err1)
	asserts.NotEqual(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
}

// TestSecretNotFound tests the Reconcile method for the following use cases
// GIVEN requests to reconcile a ConfigMap
// WHEN the Secret is not found in the cluster
// THEN Verrazzano is updated if it's listed as an override, otherwise the request is ignored
func TestSecretNotFound(t *testing.T) {
	tests := []struct {
		nsn types.NamespacedName
	}{
		{
			nsn: types.NamespacedName{Namespace: testNS, Name: testSecretName},
		},
		{
			nsn: types.NamespacedName{Namespace: testNS, Name: "test"},
		},
	}

	for i, tt := range tests {
		asserts := assert.New(t)
		cli := fake.NewClientBuilder().WithObjects(&testVZ).WithScheme(newScheme()).Build()
		config.Set(config.OperatorConfig{CloudCredentialWatchEnabled: false})
		config.TestProfilesDir = "../../manifests/profiles"
		defer func() { config.TestProfilesDir = "" }()

		request0 := newRequest(tt.nsn.Namespace, tt.nsn.Name)
		reconciler := newSecretsReconciler(cli)
		res0, err0 := reconciler.Reconcile(context.TODO(), request0)

		asserts.NoError(err0)
		asserts.Equal(false, res0.Requeue)

		vz := &vzapi.Verrazzano{}
		err1 := cli.Get(context.TODO(), types.NamespacedName{Namespace: testNS, Name: testVZName}, vz)
		asserts.NoError(err1)
		if i == 0 {
			asserts.Equal(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
		} else {
			asserts.NotEqual(int64(1), vz.Status.Components["prometheus-operator"].ReconcilingGeneration)
		}
	}

}

// TestVerrazzanoResourcesNotFound tests the Reconcile method for the following use cases
// GIVEN a request to reconcile
// WHEN no verrazzano resources are found
// THEN the secrets reconciler returns a result of ctrl.Result{}
func TestVerrazzanoResourcesNotFound(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get a list of verrazzano resources
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
			// return no resources
			return nil
		})

	// Create and make the request
	request := newRequest(vzTLSSecret.Namespace, vzTLSSecret.Name)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.NotNil(result)
	asserts.Equal(ctrl.Result{}, result)
}

// TestVerrazzanoVerrazzanoResourceBeingDeleted tests the Reconcile method for the following use cases
// GIVEN a request to reconcile
// WHEN the verrazzano resource is marked for deletion
// THEN the secrets reconciler returns a result of ctrl.Result{}
func TestVerrazzanoVerrazzanoResourceBeingDeleted(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to get a list of verrazzano resources
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, vzList *vzapi.VerrazzanoList, opts ...client.ListOption) error {
			vzList.Items = []vzapi.Verrazzano{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         constants.DefaultNamespace,
					Name:              "verrazzano",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			}}
			return nil
		})

	// Create and make the request
	request := newRequest(vzTLSSecret.Namespace, vzTLSSecret.Name)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.NotNil(result)
	asserts.Equal(ctrl.Result{}, result)
}

// TestDeletion tests the Reconcile loop for the following use case
// GIVEN a request to reconcile a Secret that qualifies as an override
// WHEN we find that it is scheduled for deletion and contains overrides finalizer
// THEN the override finalizer is removed from the Secret and Verrazzano CR is updated and request is returned without an error
func TestDeletion(t *testing.T) {
	asserts := assert.New(t)
	secret := testSecret
	secret.Finalizers = append(secret.Finalizers, constants.OverridesFinalizer)
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

// TestSecretCall tests the reconcileInstallOverrideSecret for the following use case
// GIVEN a request to reconcile a Secret
// WHEN the request namespace matches the Verrazzano CR namespace
// THEN expect a call to get the Secret
func TestSecretCall(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	expectGetSecretExists(mock, testNS, testSecretName)

	request := newRequest(testNS, testSecretName)
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.reconcileInstallOverrideSecret(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestOtherNS tests the reconcileInstallOverrideSecret for the following use case
// GIVEN a request to reconcile a Secret
// WHEN the request namespace does not match with the CR namespace
// THEN the request is ignored
func TestOtherNS(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Do not expect a call to get the Secret if it's a different namespace
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).MaxTimes(0)

	request := newRequest("test0", "test1")
	reconciler := newSecretsReconciler(mock)
	result, err := reconciler.reconcileInstallOverrideSecret(context.TODO(), request, &testVZ)
	asserts.NoError(err)
	mocker.Finish()
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

}

// mock client request to get the secret
func expectGetSecretExists(mock *mocks.MockClient, namespace string, name string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *corev1.Secret, opts ...client.GetOption) error {
			return nil
		})
}

func expectNothingForWrongSecret(mock *mocks.MockClient) {

	mock.EXPECT().
		List(gomock.Any(), &vzapi.VerrazzanoList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzanoList *vzapi.VerrazzanoList, options ...client.ListOption) error {
			return nil
		})

	// Expect no calls to get a secret
	mock.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).MaxTimes(0)

	// Expect no calls to get update
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = certv1.AddToScheme(scheme)
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
		Client:        c,
		Scheme:        scheme,
		log:           vzlog.DefaultLogger(),
		StatusUpdater: &vzstatus.FakeVerrazzanoStatusUpdater{Client: c},
	}
	return reconciler
}

// newVZ - create a Verrazzano custom resource
func newVZ() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano",
		},
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					IssuerConfig: vzapi.IssuerConfig{
						CA: &vzapi.CAIssuer{
							SecretName: constants2.DefaultVerrazzanoCASecretName,
						},
					},
					ClusterResourceNamespace: constants2.CertManagerNamespace,
				},
			},
		},
	}
}

// newCertificateWithSecret - Create a new certificate and secret that is optionally signed by a parent
func newCertificateWithSecret(issuerName string, commonName string, certName string, certNamespace string, parent *x509.Certificate) (*corev1.Secret, *certv1.Certificate, error) {
	fakeIssuerCertBytes, err := cmcommonfake.CreateFakeCertBytes(commonName, parent)
	if err != nil {
		return nil, nil, err
	}
	secret := newCertSecret(fmt.Sprintf("%s-secret", certName), certNamespace, corev1.TLSCertKey, fakeIssuerCertBytes)
	certificate := &certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: certNamespace,
		},
		Spec: certv1.CertificateSpec{
			CommonName: commonName,
			IsCA:       true,
			IssuerRef: cmmeta.ObjectReference{
				Name: issuerName,
			},
			SecretName: secret.Name,
		},
		Status: certv1.CertificateStatus{
			RenewalTime: &metav1.Time{
				// yesterday
				Time: time.Now().AddDate(0, 0, -1),
			},
		},
	}

	return secret, certificate, nil
}

func newCertSecret(name string, namespace string, certKey string, certBytes []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			certKey: certBytes,
		},
		Type: corev1.SecretTypeTLS,
	}
}
