// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"testing"

	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Kiali: &vzapi.KialiComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

var testScheme = runtime.NewScheme()

const profilesRelativePath = "../../../../manifests/profiles"

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)

	_ = vzapi.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certapiv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilKiali tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is nil
//  THEN true is returned
func TestIsEnabledNilKiali(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and managed cluster profile
//  THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and prod profile
//  THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and dev profile
//  THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestKialiPostInstallUpdateResources tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the Kiali resources already exist
//  THEN no error is returned
func TestKialiPostInstallUpdateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.KialiIngress, Namespace: constants.VerrazzanoSystemNamespace},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	authPol := &istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	fakeClient := fake.NewFakeClientWithScheme(testScheme, ingress, authPol, cert)
	err := NewComponent().PostInstall(spi.NewFakeContext(fakeClient, vz, false))
	assert.Nil(t, err)
}

// TestKialiPostInstallCreateResources tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the Kiali ingress and authpolicies don't yet exist
//  THEN no error is returned
func TestKialiPostInstallCreateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(testScheme, cert)
	err := NewComponent().PostInstall(spi.NewFakeContext(fakeClient, vz, false))
	assert.Nil(t, err)
}

// TestKialiPostUpgradeUpdateResources tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN the Kiali resources exist
//  THEN no error is returned
func TestKialiPostUpgradeUpdateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: kialiSystemName, Namespace: constants.VerrazzanoSystemNamespace},
	}
	authPol := &istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	fakeClient := fake.NewFakeClientWithScheme(testScheme, ingress, authPol)
	err := NewComponent().PostUpgrade(spi.NewFakeContext(fakeClient, vz, false))
	assert.Nil(t, err)
}

// TestPreUpgrade tests the Kiali PreUpgrade call
// GIVEN a Kiali component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../thirdparty"
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewFakeClientWithScheme(testScheme), nil, false))
	assert.NoError(t, err)
}

func getBoolPtr(b bool) *bool {
	return &b
}

func Test_kialiComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
