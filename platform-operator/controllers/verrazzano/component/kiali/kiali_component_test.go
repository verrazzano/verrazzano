// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/helm"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
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

	// +kubebuilder:scaffold:testScheme
}

// TestIsKialiReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsKialiReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewFakeClientWithScheme(testScheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      kialiSystemName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})

	assert.True(t, NewComponent().IsReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsKialiNotReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsKialiNotReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewFakeClientWithScheme(testScheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      kialiSystemName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, NewComponent().IsReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsKialiNotReadyChartNotFound tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the Kiali chart is not found
//  THEN false is returned
func TestIsKialiNotReadyChartNotFound(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewFakeClientWithScheme(testScheme)
	assert.False(t, NewComponent().IsReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath)))
}

// TestIsEnabledNilKiali tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is nil
//  THEN true is returned
func TestIsEnabledNilKiali(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and managed cluster profile
//  THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and prod profile
//  THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil and dev profile
//  THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
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
	authPol := &istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	fakeClient := fake.NewFakeClientWithScheme(testScheme, ingress, authPol)
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
	fakeClient := fake.NewFakeClientWithScheme(testScheme)
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

func getBoolPtr(b bool) *bool {
	return &b
}
