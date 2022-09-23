// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"testing"

	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../../manifests/profiles"

// TestIsEnabled tests the IsEnabled function for the Prometheus Operator component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Operator enabled
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns true
			name: "Test IsEnabled when Prometheus Operator component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Operator disabled
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Prometheus Operator component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
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
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false, profilesRelativePath)
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// TestValidateUpdate tests the Prometheus Operator ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				PrometheusOperator: &vzapi.PrometheusOperatorComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				PrometheusOperator: &vzapi.PrometheusOperatorComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	assert.Error(t, NewComponent().ValidateUpdate(&oldVZ, &newVZ))
}

// TestPostInstall tests the component PostInstall function
func TestPostInstall(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed
	// WHEN we call the PostInstall function
	// THEN no error is returned
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

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
		ObjectMeta: metav1.ObjectMeta{Name: constants.PrometheusIngress, Namespace: authproxy.ComponentNamespace},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: prometheusCertificateName, Namespace: authproxy.ComponentNamespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress, cert).Build()
	ctx := spi.NewFakeContext(client, vz, nil, false, profilesRelativePath)
	err := NewComponent().PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the component PostUpgrade function
func TestPostUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being upgraded
	// WHEN we call the PostUpgrade function
	// THEN no error is returned
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

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

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, vz, nil, false, profilesRelativePath)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}

func TestValidateInstall(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{},
			},
		},
	}
	tests := []common.ValidateInstallTest{
		{
			Name:      "NoExistingGrafana",
			WantErr:   "",
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(),
			Vz:        vz,
		},
		{
			Name:      "ExistingDeployment",
			WantErr:   istioPrometheus,
			Appsv1Cli: common.MockGetAppsV1(common.MkDep(constants.IstioSystemNamespace, istioPrometheus)),
			Corev1Cli: common.MockGetCoreV1(),
			Vz:        vz,
		},
		{
			Name:      "ExistingService",
			WantErr:   istioPrometheus,
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(common.MkSvc(constants.IstioSystemNamespace, istioPrometheus)),
			Vz:        vz,
		},
	}
	common.RunValidateInstallTest(t, NewComponent, tests...)
}
