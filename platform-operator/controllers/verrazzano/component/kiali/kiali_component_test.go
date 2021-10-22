// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestIsKialiReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsKialiReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
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

	assert.True(t, NewComponent().IsReady(spi.NewContext(zap.S(), fakeClient, &vzapi.Verrazzano{}, false)))
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

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
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
	assert.False(t, NewComponent().IsReady(spi.NewContext(zap.S(), fakeClient, &vzapi.Verrazzano{}, false)))
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

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	assert.False(t, NewComponent().IsReady(spi.NewContext(zap.S(), fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN Kiali is explicitly enabled
//  THEN true is returned
func TestIsEnabled(t *testing.T) {
	enabled := true
	assert.True(t, IsEnabled(&vzapi.KialiComponent{Enabled: &enabled}))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, IsEnabled(nil))
}

// TestIsEnabledNilEnabledFlag tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Kiali enabled flag is nil
//  THEN false is returned
func TestIsEnabledNilEnabledFlag(t *testing.T) {
	assert.False(t, IsEnabled(&vzapi.KialiComponent{}))
}

// TestPostInstallUpdateKialiIngress tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the Kiali ingress already exists
//  THEN no error is returned
func TestPostInstallUpdateKialiIngress(t *testing.T) {
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
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, ingress)
	err := NewComponent().PostInstall(spi.NewContext(zap.S(), fakeClient, vz, false))
	assert.Nil(t, err)
}

// TestPostInstallCreateKialiIngress tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the Kiali ingress doesn't yet exist
//  THEN no error is returned
func TestPostInstallCreateKialiIngress(t *testing.T) {
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
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := NewComponent().PostInstall(spi.NewContext(zap.S(), fakeClient, vz, false))
	assert.Nil(t, err)
}

// TestPostUpgradeUpdateKialiIngress tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN the Kiali ingress exists
//  THEN no error is returned
func TestPostUpgradeUpdateKialiIngress(t *testing.T) {
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
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, ingress)
	err := NewComponent().PostUpgrade(spi.NewContext(zap.S(), fakeClient, vz, false))
	assert.Nil(t, err)
}
