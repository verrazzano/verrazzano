// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	apiextv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsCertManagerOciDNSEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled with ACME and OCI DNS is configured
// THEN the function returns true
func TestIsCertManagerOciDNSEnabled(t *testing.T) {
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1client.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(crtObjectToRuntimeObject(createCertManagerCRDs()...)...).ApiextensionsV1(), nil
	}

	bt := true

	assert.False(t, NewComponent().IsEnabled(&vzapi.Verrazzano{}))

	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))

	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: &vzapi.OCI{}}
	assert.False(t, NewComponent().IsEnabled(localvz))

	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Certificate: vzapi.Certificate{
			Acme: acme,
		},
		Enabled: &bt,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: &vzapi.OCI{}}
	assert.True(t, NewComponent().IsEnabled(localvz))

	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Certificate: vzapi.Certificate{
			Acme: acme,
		},
		Enabled: &bt,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))
}

// TestIsCertManagerOciDNSDisabledNoCRDs tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerOciDNSDisabledNoCRDs(t *testing.T) {
	localvz := vz.DeepCopy()
	bf := false
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1client.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset().ApiextensionsV1(), nil
	}

	localvz.Spec.Components.CertManager.Enabled = &bf
	assert.False(t, NewComponent().IsEnabled(localvz))
}

func TestIsCertManagerOciDNSDisabled(t *testing.T) {
	localvz := vz.DeepCopy()
	bt := true
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1client.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(crtObjectToRuntimeObject(createCertManagerCRDs()...)...).ApiextensionsV1(), nil
	}

	localvz.Spec.Components.CertManager.Enabled = &bt
	assert.False(t, NewComponent().IsEnabled(localvz))
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerOciDNSPreInstallDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	err := NewComponent().PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true))
	assert.NoError(t, err)
}

// TestCertManagerPreInstallOCIDNS tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall and OCI DNS is enabled
// THEN no errors are returned and the DNS secret is set up
func TestCertManagerPreInstallOCIDNS(t *testing.T) {
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..", //since we are running inside the cert manager package, root is up 5 directories
	})
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	vz := &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: "my-verrazzano", Namespace: "default", CreationTimestamp: metav1.Now()},
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						OCIConfigSecret:        "oci",
						DNSZoneCompartmentOCID: "compartmentID",
						DNSZoneOCID:            "zoneID",
						DNSZoneName:            "zone.name.io",
					},
				},
			},
		},
	}
	err := NewComponent().PreInstall(spi.NewFakeContext(client, vz, nil, false))
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "oci", Namespace: constants.CertManagerNamespace}, secret)
	assert.NoError(t, err)
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the cert type is Acme
//	THEN no error is returned
func TestPostInstallAcme(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	// set OCI DNS secret value and create secret
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret: "ociDNSSecret",
			DNSZoneName:     "example.dns.io",
		},
	}
	_ = client.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ociDNSSecret",
			Namespace: ComponentNamespace,
		},
	})
	err := NewComponent().PostInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

// TestDryRun tests the behavior when DryRun is enabled, mainly for code coverage
// GIVEN a call to PostInstall/PostUpgrade/PreInstall
//
//	WHEN the ComponentContext has DryRun set to true
//	THEN no error is returned
func TestDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true)

	comp := certManagerOciDNSComponent{}
	assert.True(t, comp.IsReady(ctx))
}
