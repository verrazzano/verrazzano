// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerconfig

import (
	"context"
	"testing"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	common "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	profileDir    = "../../../../manifests/profiles"
	testNamespace = "testNamespace"
)

// default CA object
var ca = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: testNamespace,
}

// Default Acme object
var acme = vzapi.Acme{
	Provider:     vzapi.LetsEncrypt,
	EmailAddress: "testEmail@foo.com",
	Environment:  letsEncryptStaging,
}

// Default Verrazzano object
var defaultVZConfig = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{},
			},
		},
	},
}

// Fake certManagerConfigComponent resource for function calls
var fakeComponent = certManagerConfigComponent{}

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = apiextv1.AddToScheme(testScheme)
}

// TestIsCertManagerEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled
// THEN the function returns true
func TestIsCertManagerEnabled(t *testing.T) {
	defer func() { common.ResetAPIExtV1ClientFunc() }()
	common.SetAPIExtV1ClientFunc(getTestClient(createCertManagerCRDsRuntimeObjs()...))

	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(true)

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(createCertManagerCRDs()...).Build()
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(client, localvz, nil, false).EffectiveCR()))
}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerDisabled(t *testing.T) {
	defer func() { common.ResetAPIExtV1ClientFunc() }()
	common.SetAPIExtV1ClientFunc(getTestClient())

	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(false)
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, nil, false).EffectiveCR()))
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerPreInstallDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true))
	assert.NoError(t, err)
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall
// THEN no errors are returned
func TestCertManagerPreInstall(t *testing.T) {
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..", //since we are running inside the cert manager package, root is up 5 directories
	})
	defer func() { common.ResetAPIExtV1ClientFunc() }()
	common.SetAPIExtV1ClientFunc(getTestClient(createCertManagerCRDsRuntimeObjs()...))
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestIsCertManagerReady tests the verrazzanoCertManagerResourcesReady function
// GIVEN a call to verrazzanoCertManagerResourcesReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsCertManagerReady(t *testing.T) {
	cmCRDs := createCertManagerCRDs()
	clusterIssuer := &certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoClusterIssuerName,
		},
	}

	objects := append(cmCRDs, clusterIssuer)
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objects...).Build()

	runtimeObjects := append(createCertManagerCRDsRuntimeObjs(), clusterIssuer)
	defer func() { common.ResetAPIExtV1ClientFunc() }()
	common.SetAPIExtV1ClientFunc(getTestClient(runtimeObjects...))

	certManager := NewComponent().(certManagerConfigComponent)
	assert.True(t, certManager.verrazzanoCertManagerResourcesReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestIsCertManagerConfigNotReady tests the verrazzanoCertManagerResourcesReady function
// GIVEN a call to verrazzanoCertManagerResourcesReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerConfigNotReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	certManager := NewComponent().(certManagerConfigComponent)
	assert.False(t, certManager.verrazzanoCertManagerResourcesReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestIsCANilWithProfile tests the isCA function
// GIVEN a call to isCA
// WHEN the CertManager component is populated by the profile
// THEN true is returned
func TestIsCANilWithProfile(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	isCAValue, err := isCA(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

// TestIsCANilTrue tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate CA is populated
// THEN true is returned
func TestIsCATrue(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	isCAValue, err := isCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

func createClientFunc(caConfig vzapi.CA, cn string, otherObjs ...runtime.Object) getCoreV1ClientFuncType {
	return func(...vzlog.VerrazzanoLogger) (corev1.CoreV1Interface, error) {
		secret, err := createCertSecretNoParent(caConfig.SecretName, caConfig.ClusterResourceNamespace, cn)
		if err != nil {
			return nil, err
		}
		objs := []runtime.Object{secret}
		objs = append(objs, otherObjs...)
		return k8sfake.NewSimpleClientset(objs...).CoreV1(), nil
	}
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCAFalse(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	isCAValue, err := isCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Nil(t, err)
	assert.False(t, isCAValue)
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCABothPopulated(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	_, err := isCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Error(t, err)
}

// TestCreateCAResources tests the createOrUpdateCAResources function.
func TestCreateCAResources(t *testing.T) {
	// GIVEN that a secret with the cluster CA certificate does not exist
	// WHEN a call is made to create the CA resources
	// THEN the call succeeds and an Issuer, Certificate, and ClusterIssuer have been created
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	opResult, err := createOrUpdateCAResources(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.NoError(t, err)
	assert.Equal(t, controllerutil.OperationResultCreated, opResult)

	// validate that the Issuer, Certificate, and ClusterIssuer were created
	exists, err := issuerExists(client, caSelfSignedIssuerName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = certificateExists(client, caCertificateName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = clusterIssuerExists(client, constants.VerrazzanoClusterIssuerName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// GIVEN that a secret with the cluster CA certificate exists
	// WHEN a call is made to create the CA resources
	// THEN the call succeeds and only a ClusterIssuer has been created
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      localvz.Spec.Components.CertManager.Certificate.CA.SecretName,
			Namespace: localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace,
		},
	}
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&secret).Build()

	opResult, err = createOrUpdateCAResources(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.NoError(t, err)
	assert.Equal(t, controllerutil.OperationResultCreated, opResult)

	// validate that only the ClusterIssuer was created
	exists, err = issuerExists(client, caSelfSignedIssuerName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = certificateExists(client, caCertificateName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = clusterIssuerExists(client, constants.VerrazzanoClusterIssuerName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestRenewAllCertificatesNoCertsPresent tests the checkRenewAllCertificates function (code coverage mainly)
// GIVEN a call to checkRenewAllCertificates
//
//	WHEN No certs are found
//	THEN no error is returned
func TestRenewAllCertificatesNoCertsPresent(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	fakeContext := spi.NewFakeContext(client, defaultVZConfig, nil, false)
	assert.NoError(t, checkRenewAllCertificates(fakeContext, true))
}

// TestDeleteObject tests the deleteObject function
// GIVEN a call to deleteObject
// WHEN for an object that exists
// THEN no error is returned and the object is deleted, and that it is idempotent if called again for the same object
func TestDeleteObject(t *testing.T) {
	const name = "mysecret"
	const ns = "myns"
	secretToDelete := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}
	client := fake.NewClientBuilder().WithScheme(testScheme).
		WithObjects(
			secretToDelete,
		).Build()

	fakeContext := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false, profileDir)
	assert.NoError(t, deleteObject(fakeContext.Client(), name, ns, &v1.Secret{}))
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, &v1.Secret{})
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// Ensure this is idempotent, and no error is returned if the secret isn't found
	assert.NoError(t, deleteObject(fakeContext.Client(), name, ns, &v1.Secret{}))

}

// TestCleanupUnusedACMEResources tests the cleanupUnusedResources function
// GIVEN a call to cleanupUnusedResources
// WHEN the default issuer is configured and there are leftover ACME resources
// THEN no error is returned and default ACME CA secret is deleted without affecting the required default issuer resources
func TestCleanupUnusedACMEResources(t *testing.T) {
	vz := defaultVZConfig.DeepCopy()

	client := fake.NewClientBuilder().WithScheme(testScheme).
		WithObjects(createACMEResources()...).
		WithObjects(createClusterIssuerResources()...).
		WithObjects(createDefaultIssuerResources()...).
		Build()

	fakeContext := spi.NewFakeContext(client, vz, nil, false, profileDir)
	assert.NoError(t, cleanupUnusedResources(fakeContext, true))
	assertFound(t, client, constants.VerrazzanoClusterIssuerName, ComponentNamespace, &certv1.ClusterIssuer{})
	assertNotFound(t, client, caAcmeSecretName, ComponentNamespace, &v1.Secret{})
	assertFound(t, client, defaultCACertificateSecretName, ComponentNamespace, &v1.Secret{})
	assertFound(t, client, caCertificateName, ComponentNamespace, &certv1.Certificate{})
	assertFound(t, client, caSelfSignedIssuerName, ComponentNamespace, &certv1.Issuer{})
}

// TestCleanupUnusedDefaultCAResources tests the cleanupUnusedResources function
// GIVEN a call to cleanupUnusedResources
// WHEN the ACME issuer is configured and there are leftover default issuer resources
// THEN no error is returned and default issuer resources are deleted without affecting the required ACME resources
func TestCleanupUnusedDefaultCAResources(t *testing.T) {
	vz := defaultVZConfig.DeepCopy()
	vz.Spec.Components.CertManager.Certificate = vzapi.Certificate{}
	vz.Spec.Components.CertManager.Certificate.Acme = acme

	client := fake.NewClientBuilder().WithScheme(testScheme).
		WithObjects(createACMEResources()...).
		WithObjects(createClusterIssuerResources()...).
		WithObjects(createDefaultIssuerResources()...).
		Build()

	fakeContext := spi.NewFakeContext(client, vz, nil, false, profileDir)
	assert.NoError(t, cleanupUnusedResources(fakeContext, false))
	assertFound(t, client, caAcmeSecretName, ComponentNamespace, &v1.Secret{})
	assertFound(t, client, constants.VerrazzanoClusterIssuerName, ComponentNamespace, &certv1.ClusterIssuer{})
	assertNotFound(t, client, defaultCACertificateSecretName, ComponentNamespace, &v1.Secret{})
	assertNotFound(t, client, caCertificateName, ComponentNamespace, &certv1.Certificate{})
	assertNotFound(t, client, caSelfSignedIssuerName, ComponentNamespace, &certv1.Issuer{})
}

// TestCustomCAConfigCleanupUnusedResources tests the cleanupUnusedResources function
// GIVEN a call to cleanupUnusedResources
// WHEN a Custom CA issuer is configured and there are leftover default and ACME issuer resources
// THEN no error is returned and all leftover default and ACME issuer resources are deleted without affecting the required resources
func TestCustomCAConfigCleanupUnusedResources(t *testing.T) {
	const customCAName = "my-ca"
	const customCANamespace = "customca"

	vz := defaultVZConfig.DeepCopy()
	vz.Spec.Components.CertManager.Certificate = vzapi.Certificate{}
	vz.Spec.Components.CertManager.Certificate.CA = vzapi.CA{
		SecretName:               customCAName,
		ClusterResourceNamespace: customCANamespace,
	}

	client := fake.NewClientBuilder().WithScheme(testScheme).
		WithObjects(createClusterIssuerResources()...).
		WithObjects(createCustomCAResources(customCAName, customCANamespace)...).
		WithObjects(createACMEResources()...).
		WithObjects(createDefaultIssuerResources()...).
		Build()

	fakeContext := spi.NewFakeContext(client, vz, nil, false, profileDir)
	assert.NoError(t, cleanupUnusedResources(fakeContext, true))
	assertFound(t, client, customCAName, customCANamespace, &v1.Secret{})
	assertFound(t, client, constants.VerrazzanoClusterIssuerName, ComponentNamespace, &certv1.ClusterIssuer{})
	assertNotFound(t, client, caAcmeSecretName, ComponentNamespace, &v1.Secret{})
	assertNotFound(t, client, defaultCACertificateSecretName, ComponentNamespace, &v1.Secret{})
	assertNotFound(t, client, caCertificateName, ComponentNamespace, &certv1.Certificate{})
	assertNotFound(t, client, caSelfSignedIssuerName, ComponentNamespace, &certv1.Issuer{})
}

// TestUninstallCertManager tests the cert-manager uninstall process
// GIVEN a call to uninstallVerrazzanoCertManagerResources
// WHEN the objects exist in the cluster
// THEN no error is returned and all objects are deleted
func TestUninstallCertManager(t *testing.T) {
	vz := defaultVZConfig.DeepCopy()

	certNS := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.CertManagerNamespace,
		},
	}
	vzCI := certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoClusterIssuerName,
		},
	}
	ssIssuer := certv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      caSelfSignedIssuerName,
		},
	}
	caCert := certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      caCertificateName,
		},
	}
	caSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      defaultCACertificateSecretName,
		},
	}
	caAcmeSec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      caAcmeSecretName,
		},
	}

	tests := []struct {
		name    string
		objects []clipkg.Object
	}{
		{
			name: "test empty cluster",
		},
		{
			name: "test cluster issuer",
			objects: []clipkg.Object{
				&certNS,
				&vzCI,
			},
		},
		{
			name: "test issuer",
			objects: []clipkg.Object{
				&certNS,
				&vzCI,
				&ssIssuer,
			},
		},
		{
			name: "test ca cert",
			objects: []clipkg.Object{
				&certNS,
				&vzCI,
				&ssIssuer,
				&caCert,
			},
		},
		{
			name: "test ca secret",
			objects: []clipkg.Object{
				&certNS,
				&vzCI,
				&ssIssuer,
				&caCert,
				&caSecret,
			},
		},
		{
			name: "test ca acme secret",
			objects: []clipkg.Object{
				&certNS,
				&vzCI,
				&ssIssuer,
				&caCert,
				&caSecret,
				&caAcmeSec,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(tt.objects...).Build()
			fakeContext := spi.NewFakeContext(c, vz, nil, false, profileDir)
			err := uninstallVerrazzanoCertManagerResources(fakeContext)
			assert.NoError(t, err)
			// expect the Namespace to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoClusterIssuerName, Namespace: vzconst.DefaultNamespace}, &certv1.ClusterIssuer{})
			assert.Error(t, err, "Expected the ClusterIssuer %s to be deleted", ComponentNamespace)
			// expect the Namespace to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: caSelfSignedIssuerName, Namespace: ComponentNamespace}, &certv1.Issuer{})
			assert.Error(t, err, "Expected the Issuer %s to be deleted", ComponentNamespace)
			// expect the Namespace to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: caCertificateName, Namespace: ComponentNamespace}, &certv1.Certificate{})
			assert.Error(t, err, "Expected the Certificate %s to be deleted", ComponentNamespace)
			// expect the Namespace to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: defaultCACertificateSecretName, Namespace: ComponentNamespace}, &v1.Secret{})
			assert.Error(t, err, "Expected the Secret %s to be deleted", ComponentNamespace)
			// expect the Namespace to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: caAcmeSecretName, Namespace: ComponentNamespace}, &v1.Secret{})
			assert.Error(t, err, "Expected the Secret %s to be deleted", ComponentNamespace)
		})
	}
}

func assertNotFound(t *testing.T, client clipkg.WithWatch, name string, namespace string, obj clipkg.Object) {
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, obj)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func assertFound(t *testing.T, client clipkg.WithWatch, name string, namespace string, obj clipkg.Object) {
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, obj)
	assert.NoError(t, err)
}

func createACMEResources() []clipkg.Object {
	acmeSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: caAcmeSecretName, Namespace: ComponentNamespace},
	}
	return []clipkg.Object{acmeSecret}
}

func createCustomCAResources(name string, namespace string) []clipkg.Object {
	customcA := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return []clipkg.Object{customcA}
}

func createClusterIssuerResources() []clipkg.Object {
	customcA := &certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: constants.VerrazzanoClusterIssuerName, Namespace: ComponentNamespace},
	}
	return []clipkg.Object{customcA}
}

func createDefaultIssuerResources() []clipkg.Object {
	secretToDelete := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: defaultCACertificateSecretName, Namespace: ComponentNamespace},
	}
	selfSignedCACert := &certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: caCertificateName, Namespace: ComponentNamespace},
	}
	selfSignedIssuer := &certv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{Name: caSelfSignedIssuerName, Namespace: ComponentNamespace},
	}
	return []clipkg.Object{secretToDelete, selfSignedCACert, selfSignedIssuer}
}

// issuerExists returns true if the Issuer with the name and namespace exists.
func issuerExists(client clipkg.Client, name string, namespace string) (bool, error) {
	issuer := certv1.Issuer{}
	if err := client.Get(context.TODO(), clipkg.ObjectKey{Name: name, Namespace: namespace}, &issuer); err != nil {
		return false, clipkg.IgnoreNotFound(err)
	}
	return true, nil
}

// clusterIssuerExists returns true if the ClusterIssuer with the name exists.
func clusterIssuerExists(client clipkg.Client, name string) (bool, error) {
	clusterIssuer := certv1.ClusterIssuer{}
	if err := client.Get(context.TODO(), clipkg.ObjectKey{Name: name}, &clusterIssuer); err != nil {
		return false, clipkg.IgnoreNotFound(err)
	}
	return true, nil
}

// certificateExists returns true if the Certificate with the name and namespace exists.
func certificateExists(client clipkg.Client, name string, namespace string) (bool, error) {
	cert := certv1.Certificate{}
	if err := client.Get(context.TODO(), clipkg.ObjectKey{Name: name, Namespace: namespace}, &cert); err != nil {
		return false, clipkg.IgnoreNotFound(err)
	}
	return true, nil
}

func createCertManagerCRDsRuntimeObjs() []runtime.Object {
	var cmCRDs []runtime.Object
	for _, crd := range common.GetRequiredCertManagerCRDNames() {
		cmCRDs = append(cmCRDs, newCRD(crd))
	}
	return cmCRDs
}

func createCertManagerCRDs() []clipkg.Object {
	var cmCRDs []clipkg.Object
	for _, crd := range common.GetRequiredCertManagerCRDNames() {
		cmCRDs = append(cmCRDs, newCRD(crd))
	}
	return cmCRDs
}

func newCRD(name string) clipkg.Object {
	crd := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return crd
}

// Create a bool pointer
func getBoolPtr(b bool) *bool {
	return &b
}
