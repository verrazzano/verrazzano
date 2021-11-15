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
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testUtilDir = "./test_utils/"
	utilDir     = "./utils/"
)

var ca = vzapi.CA{
	SecretName: "testSecret",
	ClusterResourceNamespace: namespace,
}

var acme = vzapi.Acme{
	Provider: "testProvider",
	EmailAddress: "testEmail",
	Environment: "myenv",
}

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

var fakeComponent = certManagerComponent{}

// TestIsCertManagerEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
//  WHEN I have cert-manager enabled
//  THEN the function returns true
func TestIsCertManagerEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(true)
	assert.True(t, IsCertManagerEnabled(spi.NewContext(zap.S(), nil, localvz, false)))
}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
//  WHEN I have cert-manager disabled
//  THEN the function returns false
func TestIsCertManagerDisabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(false)
	assert.False(t, IsCertManagerEnabled(spi.NewContext(zap.S(), nil, localvz, false)))
}


// TestAppendCertManagerOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass a VZ spec with defaults
//  THEN the values created properly
func TestAppendCertManagerOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, vz, false), ComponentName, namespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 0)
}

// TestAppendCertManagerOverridesWithInstallArgs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//  WHEN I pass a VZ spec with install args
//  THEN the values created properly
func TestAppendCertManagerOverridesWithInstallArgs(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	kvs, err := AppendOverrides(spi.NewContext(zap.S(), nil, localvz, false), ComponentName, namespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
//  WHEN I call PreInstall
//  THEN no errors are returned
func TestCertManagerPreInstallDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := fakeComponent.PreInstall(spi.NewContext(zap.S(), client, &vzapi.Verrazzano{}, true))
	assert.NoError(t, err)
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
//  WHEN I call PreInstall
//  THEN no errors are returned
func TestCertManagerPreInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	setBashFunc(fakeBash)
	err := fakeComponent.PreInstall(spi.NewContext(zap.S(), client, &vzapi.Verrazzano{}, false))
	assert.NoError(t, err)
}


// TestIsCertManagerReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsCertManagerReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(certManagerDeploymentName, true),
		newDeployment(cainjectorDeploymentName, true),
		newDeployment(webhookDeploymentName, true),
	)
	assert.True(t, IsReady(spi.NewContext(zap.S(), fakeClient, nil, false), ComponentName, namespace))
}

// TestIsCertManagerNotReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object does not have enough replicas available
//  THEN false is returned
func TestIsCertManagerNotReady(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(certManagerDeploymentName, false),
		newDeployment(cainjectorDeploymentName, false),
		newDeployment(webhookDeploymentName, false),
	)
	assert.False(t, IsReady(spi.NewContext(zap.S(), fakeClient, nil, false), ComponentName, namespace))
}

// TestGetCertificateConfigNil tests the getCertificateConfig function
// GIVEN a call to getCertificateConfig
//  WHEN the CertManager component is nil
//  THEN the CA field is added
func TestGetCertificateConfigNil(t *testing.T) {
	assert.Equal(t, vzapi.CA{
			ClusterResourceNamespace: defaultCAClusterResourceName,
			SecretName: defaultCASecretName}, *getCertificateConfig(vz).CA)
}

// TestGetCertificateConfigCA tests the getCertificateConfig function
// GIVEN a call to getCertificateConfig
//  WHEN the certificate is of type CA
//  THEN the CA field is added
func TestGetCertificateConfigCA(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	assert.Equal(t, Certificate{CA: &ca}, getCertificateConfig(localvz))
}

// TestGetCertificateConfigAcme tests the getCertificateConfig function
// GIVEN a call to getCertificateConfig
//  WHEN the deployment object does not have enough replicas available
//  THEN false is returned
func TestGetCertificateConfigAcme(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	assert.Equal(t, Certificate{ACME: &acme}, getCertificateConfig(localvz))
}

// TestPostInstallCA tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the cert type is CA
//  THEN no error is returned
func TestPostInstallCA(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	scheme := k8scheme.Scheme
	certv1.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)
	err := fakeComponent.PostInstall(spi.NewContext(zap.S(), client, localvz, false))
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
	client.Create(context.TODO(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-container-registry",
			Namespace: namespace,
		},
	})
	err := fakeComponent.PostInstall(spi.NewContext(zap.S(), client, localvz, false))
	assert.NoError(t, err)
}

// fakeBash verifies that the correct parameter values are passed to upgrade
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

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

func getBoolPtr(b bool) *bool {
	return &b
}