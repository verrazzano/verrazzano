// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	fake2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common/fake"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	bt = true
	bf = false
)

// TestIsCertManagerOciDNSEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled with LetsEncrypt and OCI DNS is configured
// THEN the function returns true
func TestIsCertManagerOciDNSEnabled(t *testing.T) {

	leIssuer := vzapi.IssuerConfig{
		LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
			EmailAddress: acme.EmailAddress,
			Environment:  acme.Environment,
		},
	}

	// Default is false if nothing is configured
	assert.False(t, NewComponent().IsEnabled(&vzapi.Verrazzano{}))

	// True if explicitly enabled
	assert.True(t, NewComponent().IsEnabled(&vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
					Enabled: &bt,
				},
			},
		},
	}))

	// False if not explicitly enabled, Issuer is enabled, no OCI DNS/LE
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))

	// False - not explicitly enabled, OCI DNS enabled, Issuer is enabled, but no LE certs
	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: &vzapi.OCI{}}
	assert.False(t, NewComponent().IsEnabled(localvz))

	// True - not explicitly enabled, OCI DNS enabled, LE certs enabled, Issuer is enabled
	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{
		IssuerConfig: leIssuer,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: &vzapi.OCI{}}
	assert.True(t, certManagerWebhookOCIComponent{}.IsEnabled(localvz))

	// False - not explicitly enabled, Issuer is enabled, but no OCI DNS or LE certs
	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{
		IssuerConfig: leIssuer,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))

	// False - not explicitly enabled, Issuer is enabled, but no OCI DNS or LE certs
	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{
		IssuerConfig: leIssuer,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))

	// False - not explicitly enabled, Issuer is enabled, but no OCI DNS or LE certs
	localvz = vz.DeepCopy()
	localvz.Spec.Components.CertManager = &vzapi.CertManagerComponent{
		Enabled: &bt,
	}
	localvz.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{
		IssuerConfig: leIssuer,
	}
	assert.False(t, NewComponent().IsEnabled(localvz))
}

// TestMonitorOverrides tests MonitorOverrides
// GIVEN a call to MonitorOverrides
//
//	THEN true is returned when the webhook component exists or the MonitorChanges flag is true, false otherwise
func TestMonitorOverrides(t *testing.T) {
	asserts := assert.New(t)

	ctx := spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false)
	comp := certManagerWebhookOCIComponent{}
	asserts.False(comp.MonitorOverrides(ctx))

	localVZ := vz.DeepCopy()
	localVZ.Spec.Components.CertManagerWebhookOCI = &vzapi.CertManagerWebhookOCIComponent{}
	ctx = spi.NewFakeContext(nil, localVZ, nil, false)
	asserts.True(comp.MonitorOverrides(ctx))

	localVZ.Spec.Components.CertManagerWebhookOCI = &vzapi.CertManagerWebhookOCIComponent{
		InstallOverrides: vzapi.InstallOverrides{MonitorChanges: fake2.GetBoolPtr(true)},
	}
	ctx = spi.NewFakeContext(nil, localVZ, nil, false)
	asserts.True(comp.MonitorOverrides(ctx))

	localVZ.Spec.Components.CertManagerWebhookOCI = &vzapi.CertManagerWebhookOCIComponent{
		InstallOverrides: vzapi.InstallOverrides{MonitorChanges: fake2.GetBoolPtr(false)},
	}
	ctx = spi.NewFakeContext(nil, localVZ, nil, false)
	asserts.False(comp.MonitorOverrides(ctx))

	localVZ.Spec.Components.CertManagerWebhookOCI = &vzapi.CertManagerWebhookOCIComponent{
		InstallOverrides: vzapi.InstallOverrides{},
	}
	ctx = spi.NewFakeContext(nil, localVZ, nil, false)
	asserts.True(comp.MonitorOverrides(ctx))
}

type validationTestStruct struct {
	name    string
	old     *vzapi.Verrazzano
	new     *vzapi.Verrazzano
	wantErr bool
}

var simpleValidationTests = []validationTestStruct{
	{
		name:    "Implicitly Disabled Empty CR",
		old:     &vzapi.Verrazzano{},
		new:     &vzapi.Verrazzano{},
		wantErr: false,
	},
	{
		name: "ExplicitlyEnabled",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
						Enabled: &bt,
					},
				},
			},
		},
		wantErr: false,
	},
	{
		name: "Issuer and OCI DNS",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					ClusterIssuer: &vzapi.ClusterIssuerComponent{
						Enabled: &bf,
					},
					DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
				},
			},
		},
		wantErr: false,
	},
	{
		name: "Issuer Enabled with OCI DNS",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					ClusterIssuer: &vzapi.ClusterIssuerComponent{
						Enabled: &bt,
					},
					DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
				},
			},
		},
		wantErr: false,
	},
	{
		name: "Issuer Enabled with OCI DNS and LetsEncrypt",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					ClusterIssuer: &vzapi.ClusterIssuerComponent{
						Enabled: &bt,
						IssuerConfig: vzapi.IssuerConfig{
							LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{},
						},
					},
					DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
				},
			},
		},
		wantErr: false,
	},
	//{
	//	name: "Webhook disabled Issuer Enabled with OCI DNS and LetsEncrypt",
	//	old:  &vzapi.Verrazzano{},
	//	new: &vzapi.Verrazzano{
	//		Spec: vzapi.VerrazzanoSpec{
	//			Components: vzapi.ComponentSpec{
	//				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
	//					Enabled: &bf,
	//				},
	//				ClusterIssuer: &vzapi.ClusterIssuerComponent{
	//					Enabled: &bt,
	//					IssuerConfig: vzapi.IssuerConfig{
	//						LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{},
	//					},
	//				},
	//				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
	//			},
	//		},
	//	},
	//	wantErr: true,
	//},
	//{
	//	name: "Webhook disabled Issuer Enabled with OCI DNS and LetsEncrypt",
	//	old:  &vzapi.Verrazzano{},
	//	new: &vzapi.Verrazzano{
	//		Spec: vzapi.VerrazzanoSpec{
	//			Components: vzapi.ComponentSpec{
	//				CertManagerWebhookOCI: &vzapi.CertManagerWebhookOCIComponent{
	//					Enabled: &bf,
	//				},
	//				ClusterIssuer: &vzapi.ClusterIssuerComponent{
	//					Enabled: &bt,
	//					IssuerConfig: vzapi.IssuerConfig{
	//						LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{},
	//					},
	//				},
	//				DNS: &vzapi.DNSComponent{OCI: &vzapi.OCI{}},
	//			},
	//		},
	//	},
	//	wantErr: true,
	//},
}

// TestValidateInstall tests the ValidateInstall function
// GIVEN a call to ValidateInstall
//
//	WHEN for various webhook configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateInstall(t *testing.T) {
	runValidationTests(t, false)
}

// TestValidateUpdate tests the ValidateUpdate function
// GIVEN a call to ValidateUpdate
//
//	WHEN for various webhook configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateUpdate(t *testing.T) {
	runValidationTests(t, true)
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

	comp := certManagerWebhookOCIComponent{}
	assert.True(t, comp.IsReady(ctx))
}

func runValidationTests(t *testing.T, isUpdate bool) {
	for _, tt := range simpleValidationTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Cert Manager Namespace already exists" && isUpdate { // will throw error only during installation
				tt.wantErr = false
			}
			runValidationTest(t, tt, isUpdate, NewComponent())
		})
	}
}

func runValidationTest(t *testing.T, tt validationTestStruct, isUpdate bool, c spi.Component) {
	//	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	if isUpdate {
		if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1New := &v1beta1.Verrazzano{}
		v1beta1Old := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1New)
		assert.NoError(t, err)
		err = tt.old.ConvertTo(v1beta1Old)
		assert.NoError(t, err)
		if err := c.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}

	} else {
		if err := c.ValidateInstall(tt.new); (err != nil) != tt.wantErr {
			t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1Vz := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1Vz)
		assert.NoError(t, err)
		if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != tt.wantErr {
			t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}
