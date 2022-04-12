// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/runtime"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
var vz = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{},
			},
		},
	},
}

// Fake certManagerComponent resource for function calls
var fakeComponent = certManagerComponent{}

var testScheme = runtime.NewScheme()

func init() {
	k8scheme.AddToScheme(testScheme)
	certv1.AddToScheme(testScheme)
	vzapi.AddToScheme(testScheme)
}

// TestIsCertManagerEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled
// THEN the function returns true
func TestIsCertManagerEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(true)
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false).EffectiveCR()))
}

// TestIsOCIDNS tests whether the Effective CR is using OCI DNS
// GIVEN a call to isOCIDNS
// WHEN OCI DNS is specified in the Verrazzano Spec
// THEN isOCIDNS should return true
func TestIsOCIDNS(t *testing.T) {
	var tests = []struct {
		name   string
		vz     *vzapi.Verrazzano
		ocidns bool
	}{
		{
			"shouldn't be enabled when nil",
			&vzapi.Verrazzano{},
			false,
		},
		{
			"should be enabled when OCI DNS present",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{},
						},
					},
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ocidns, isOCIDNS(tt.vz))
		})
	}
}

// TestWriteCRD tests writing out the OCI DNS metadata to CertManager CRDs
// GIVEN a call to writeCRD
// WHEN the input file exists
// THEN the output should have ocidns added.
func TestWriteCRD(t *testing.T) {
	inputFile := "../../../../thirdparty/manifests/cert-manager/cert-manager.crds.yaml"
	outputFile := "../../../../thirdparty/manifests/cert-manager/output.crd.yaml"
	err := writeCRD(inputFile, outputFile, true)
	assert.NoError(t, err)
}

// TestCleanTempFiles tests cleaning up temp files
// GIVEN a call to cleanTempFiles
// WHEN a file is not found
// THEN cleanTempFiles should return an error
func TestCleanTempFiles(t *testing.T) {
	assert.Error(t, cleanTempFiles("blahblah"))
}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerDisabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(false)
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false).EffectiveCR()))
}

// TestAppendCertManagerOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendCertManagerOverrides(t *testing.T) {
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestAppendCertManagerOverridesWithInstallArgs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with install args
// THEN the values created properly
func TestAppendCertManagerOverridesWithInstallArgs(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "vz-cn")
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
	assert.Contains(t, kvs, bom.KeyValue{Key: clusterResourceNamespaceKey, Value: testNamespace})
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerPreInstallDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, true))
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
	client := fake.NewFakeClientWithScheme(testScheme)
	setBashFunc(fakeBash)
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, false))
	assert.NoError(t, err)
}

// TestIsCertManagerReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsCertManagerReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme,
		newDeployment(certManagerDeploymentName, map[string]string{"app": certManagerDeploymentName}, true),
		newDeployment(cainjectorDeploymentName, map[string]string{"app": "cainjector"}, true),
		newDeployment(webhookDeploymentName, map[string]string{"app": "webhook"}, true),
	)
	assert.True(t, isCertManagerReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsCertManagerNotReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerNotReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme,
		newDeployment(certManagerDeploymentName, map[string]string{"app": certManagerDeploymentName}, false),
		newDeployment(cainjectorDeploymentName, map[string]string{"app": "cainjector"}, false),
		newDeployment(webhookDeploymentName, map[string]string{"app": "webhook"}, false),
	)
	assert.False(t, isCertManagerReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsCANil tests the isCA function
// GIVEN a call to isCA
// WHEN the CertManager component is populated by the profile
// THEN true is returned
func TestIsCANilWithProfile(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	isCAValue, err := isCA(spi.NewFakeContext(client, &vzapi.Verrazzano{}, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

// TestIsCANilTrue tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate CA is populated
// THEN true is returned
func TestIsCATrue(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "vz-cn")

	client := fake.NewFakeClientWithScheme(testScheme)
	isCAValue, err := isCA(spi.NewFakeContext(client, localvz, false, profileDir))
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
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(testScheme)
	isCAValue, err := isCA(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.Nil(t, err)
	assert.False(t, isCAValue)
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCABothPopulated(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(testScheme)
	_, err := isCA(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.Error(t, err)
}

// TestCreateCAResources tests the createOrUpdateCAResources function.
func TestCreateCAResources(t *testing.T) {
	// GIVEN that a secret with the cluster CA certificate does not exist
	// WHEN a call is made to create the CA resources
	// THEN the call succeeds and an Issuer, Certificate, and ClusterIssuer have been created
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	client := fake.NewFakeClientWithScheme(testScheme)

	opResult, err := createOrUpdateCAResources(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.NoError(t, err)
	assert.Equal(t, controllerutil.OperationResultCreated, opResult)

	// validate that the Issuer, Certificate, and ClusterIssuer were created
	exists, err := issuerExists(client, caSelfSignedIssuerName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = certificateExists(client, caCertificateName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = clusterIssuerExists(client, verrazzanoClusterIssuerName)
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
	client = fake.NewFakeClientWithScheme(testScheme, &secret)

	opResult, err = createOrUpdateCAResources(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.NoError(t, err)
	assert.Equal(t, controllerutil.OperationResultCreated, opResult)

	// validate that only the ClusterIssuer was created
	exists, err = issuerExists(client, caSelfSignedIssuerName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = certificateExists(client, caCertificateName, localvz.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace)
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = clusterIssuerExists(client, verrazzanoClusterIssuerName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestRenewAllCertificatesNoCertsPresent tests the checkRenewAllCertificates function (code coverage mainly)
// GIVEN a call to checkRenewAllCertificates
//  WHEN No certs are found
//  THEN no error is returned
func TestRenewAllCertificatesNoCertsPresent(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	fakeContext := spi.NewFakeContext(client, vz, false)
	assert.NoError(t, checkRenewAllCertificates(fakeContext, true))
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

// fakeBash verifies that the correct parameter values are passed to upgrade
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

// Create a new deployment object for testing
func newDeployment(name string, labels map[string]string, updated bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    labels,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	}

	if !updated {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 1,
			UpdatedReplicas:   0,
		}
	}
	return deployment
}

// Create a bool pointer
func getBoolPtr(b bool) *bool {
	return &b
}
