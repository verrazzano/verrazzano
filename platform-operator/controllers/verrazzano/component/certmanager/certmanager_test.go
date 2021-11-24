// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	profileDir = "../../../../manifests/profiles"
)

// default CA object
var ca = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: namespace,
}

// Default Acme object
var acme = vzapi.Acme{
	Provider:     "testProvider",
	EmailAddress: "testEmail",
	Environment:  "myenv",
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

// TestIsCertManagerEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled
// THEN the function returns true
func TestIsCertManagerEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(true)
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false)))
}

// TestWriteOCICRD tests writing out the OCI DNS metadata to CertManager CRDs
// GIVEN a call to writeOCICRD
// WHEN the input file exists
// THEN the outfile should have ocidns added. there should be 7 CRDs in the manifest directory,
// 6 generated files plus the 1 existing file
func TestWriteOCICRD(t *testing.T) {
	inputFile := "../../../../thirdparty/manifests/cert-manager/cert-manager.crds.yaml"
	outputFile := "../../../../thirdparty/manifests/cert-manager/cert-manager-ocidns.crds.yaml"
	err := writeOCICRD(inputFile, outputFile)
	assert.NoError(t, err)

	files := 7
	dir, err := os.ReadDir("../../../../thirdparty/manifests/cert-manager/")
	assert.NoError(t, err)
	assert.Equal(t, files, len(dir))
}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerDisabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(false)
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false)))
}

// TestAppendCertManagerOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendCertManagerOverrides(t *testing.T) {
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profileDir), ComponentName, namespace, "", []bom.KeyValue{})
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
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, false), ComponentName, namespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerPreInstallDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
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
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	setBashFunc(fakeBash)
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, false))
	assert.NoError(t, err)
}

// TestIsCertManagerReady tests the IsReady function
// GIVEN a call to IsReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsCertManagerReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(certManagerDeploymentName, true),
		newDeployment(cainjectorDeploymentName, true),
		newDeployment(webhookDeploymentName, true),
	)
	assert.True(t, fakeComponent.IsReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsCertManagerNotReady tests the IsReady function
// GIVEN a call to IsReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerNotReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(certManagerDeploymentName, false),
		newDeployment(cainjectorDeploymentName, false),
		newDeployment(webhookDeploymentName, false),
	)
	assert.False(t, fakeComponent.IsReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsCANil tests the isCA function
// GIVEN a call to isCA
// WHEN the CertManager component is nil
// THEN an error is returned
func TestIsCANil(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	_, err := isCA(spi.NewFakeContext(client, &vzapi.Verrazzano{}, false))
	assert.Error(t, err)
}

// TestIsCANil tests the isCA function
// GIVEN a call to isCA
// WHEN the CertManager component is populated by the profile
// THEN true is returned
func TestIsCANilWithProfile(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
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
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	isCAValue, err := isCA(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCAFalse(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
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
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	_, err := isCA(spi.NewFakeContext(client, localvz, false, profileDir))
	assert.Error(t, err)
}

// TestPostInstallCA tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the cert type is CA
//  THEN no error is returned
func TestPostInstallCA(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	scheme := k8scheme.Scheme
	// Add cert-manager CRDs to scheme
	certv1.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the cert type is Acme
//  THEN no error is returned
func TestPostInstallAcme(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	// set OCI DNS secret value and create secret
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret: "ociDNSSecret",
			DNSZoneName:     "example.dns.io",
		},
	}
	client.Create(context.TODO(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ociDNSSecret",
			Namespace: namespace,
		},
	})
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

// fakeBash verifies that the correct parameter values are passed to upgrade
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

// Create a new deployment object for testing
func newDeployment(name string, ready bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	}

	if !ready {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		}
	}
	return deployment
}

// Create a bool pointer
func getBoolPtr(b bool) *bool {
	return &b
}
