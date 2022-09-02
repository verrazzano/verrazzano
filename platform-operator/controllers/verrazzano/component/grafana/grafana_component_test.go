// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles/v1alpha1"

var (
	falseValue = false
	trueValue  = true
)

// TestIsEnabled tests the IsEnabled function for the Grafana component
func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsEnabled on the Grafana component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana enabled
			// WHEN we call IsEnabled on the Grafana component
			// THEN the call returns true
			name: "Test IsEnabled when Grafana component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana disabled
			// WHEN we call IsEnabled on the Grafana component
			// THEN the call returns false
			name: "Test IsEnabled when Grafana component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false)
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// TestGetIngressNames tests getting Grafana ingress names
func TestGetIngressNames(t *testing.T) {
	grafanaIngressNames := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      constants.GrafanaIngress,
	}
	tests := []struct {
		name      string
		actualCR  vzapi.Verrazzano
		ingresses []types.NamespacedName
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call GetIngressNames on the Grafana component
			// THEN we expect to find the Grafana ingress
			name:      "Test GetIngressNames when using default Verrazzano CR",
			actualCR:  vzapi.Verrazzano{},
			ingresses: []types.NamespacedName{grafanaIngressNames},
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana and Nginx components enabled
			// WHEN we call GetIngressNames on the Grafana component
			// THEN we expect to find the Grafana ingress
			name: "Test GetIngressNames when Grafana and Nginx components set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &trueValue,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			ingresses: []types.NamespacedName{grafanaIngressNames},
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana component enabled and Nginx disabled
			// WHEN we call GetIngressNames on the Grafana component
			// THEN we do not expect to find the Grafana ingress
			name: "Test GetIngressNames when Grafana component set to enabled and Nginx is disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &trueValue,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			ingresses: nil,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false)
			assert.Equal(t, tt.ingresses, NewComponent().GetIngressNames(ctx))
		})
	}
}

// TestGetCertificateNames tests getting Grafana TLS certificate names
func TestGetCertificateNames(t *testing.T) {
	grafanaCertNames := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      grafanaCertificateName,
	}
	tests := []struct {
		name     string
		actualCR vzapi.Verrazzano
		certs    []types.NamespacedName
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call GetCertificateNames on the Grafana component
			// THEN we expect to find the Grafana certificate name
			name:     "Test GetCertificateNames when using default Verrazzano CR",
			actualCR: vzapi.Verrazzano{},
			certs:    []types.NamespacedName{grafanaCertNames},
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana and Nginx components enabled
			// WHEN we call GetCertificateNames on the Grafana component
			// THEN we expect to find the Grafana certificate name
			name: "Test GetCertificateNames when Grafana and Nginx components set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &trueValue,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			certs: []types.NamespacedName{grafanaCertNames},
		},
		{
			// GIVEN a Verrazzano custom resource with the Grafana component enabled and Nginx disabled
			// WHEN we call GetCertificateNames on the Grafana component
			// THEN we do not expect to find the Grafana certificate name
			name: "Test GetCertificateNames when Grafana component set to enabled and Nginx is disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &trueValue,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			certs: nil,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false)
			assert.Equal(t, tt.certs, NewComponent().GetCertificateNames(ctx))
		})
	}
}

// TestPreInstall tests the Grafana component PreInstall function
func TestPreInstall(t *testing.T) {
	// GIVEN Verrazzano is being installed
	// WHEN the Grafana component PreInstall function is called
	// THEN the function succeeds and the Grafana admin secret has been created
	client := fake.NewFakeClientWithScheme(testScheme)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(client, vz, nil, false)
	err := NewComponent().PreInstall(ctx)
	assert.NoError(t, err)

	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.GrafanaSecret, Namespace: globalconst.VerrazzanoSystemNamespace}, &v1.Secret{})
	assert.NoError(t, err)
}

// TestPreUpgrade tests the Grafana component PreUpgrade function
func TestPreUpgrade(t *testing.T) {
	// GIVEN Verrazzano is being upgraded
	// WHEN the Grafana component PreUpgrade function is called
	// THEN the function succeeds and the Grafana admin secret has been created
	client := fake.NewFakeClientWithScheme(testScheme)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(client, vz, nil, false)
	err := NewComponent().PreUpgrade(ctx)
	assert.NoError(t, err)

	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.GrafanaSecret, Namespace: globalconst.VerrazzanoSystemNamespace}, &v1.Secret{})
	assert.NoError(t, err)
}

// TestInstall tests the Grafana component Install function
func TestInstall(t *testing.T) {
	// GIVEN a Verrazzano CR with Grafana enabled
	// WHEN the Grafana component Install function is called
	// THEN the system dashboards configmap is created
	// AND the VMI instance is created with the expected Grafana config
	testInstallOrUpgrade(t, NewComponent().Install)
}

func TestUpgrade(t *testing.T) {
	// GIVEN a Verrazzano CR with Grafana enabled
	// WHEN the Grafana component Upgrade function is called
	// THEN the system dashboards configmap is created
	// AND the VMI instance is created with the expected Grafana config
	testInstallOrUpgrade(t, NewComponent().Upgrade)
}

// testInstallOrUpgrade tests both the Grafana component Install and Update functions
func testInstallOrUpgrade(t *testing.T, installOrUpgradeFunc func(spi.ComponentContext) error) {
	client := fake.NewFakeClientWithScheme(testScheme)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{
						Suffix: "unittestdomain",
					},
				},
			},
		},
	}
	ctx := spi.NewFakeContext(client, vz, nil, false, profilesRelativePath)
	err := installOrUpgradeFunc(ctx)
	assert.NoError(t, err)

	// make sure the system dashboards configmap was created
	dashboardsConfigMap := &v1.ConfigMap{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "system-dashboards", Namespace: globalconst.VerrazzanoSystemNamespace}, dashboardsConfigMap)
	assert.NoError(t, err)
	assert.Len(t, dashboardsConfigMap.Data, len(dashboardList))

	// make sure the VMI was created and the Grafana config is as expected
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "system", Namespace: globalconst.VerrazzanoSystemNamespace}, vmi)
	assert.NoError(t, err)
	assert.True(t, vmi.Spec.Grafana.Enabled)
	assert.Equal(t, vmi.Spec.Grafana.DashboardsConfigMap, "system-dashboards")
}

// TestValidateUpdate tests the Grafana component ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	// GIVEN an old VZ with Grafana enabled and a new VZ with Grafana disabled
	// WHEN we call the ValidateUpdate function
	// THEN the function returns an error
	oldVz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	newVz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	assert.Error(t, NewComponent().ValidateUpdate(oldVz, newVz))

	// GIVEN an old VZ with Grafana enabled and a new VZ with Grafana enabled
	// WHEN we call the ValidateUpdate function
	// THEN the function does not return an error
	newVz.Spec.Components.Grafana.Enabled = &trueValue
	assert.NoError(t, NewComponent().ValidateUpdate(oldVz, newVz))
}
