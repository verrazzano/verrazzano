// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"context"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	v1 "k8s.io/api/core/v1"
	nw "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

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

// TestName tests the Name function for the Grafana component
// GIVEN a component
// WHEN we call Name() on the Grafana component
// THEN the call returns the name of that component
func TestName(t *testing.T) {
	assert.Equal(t, "grafana", NewComponent().Name())
}

// TestNamespace tests the Namespace function for the Grafana component
// GIVEN a component
// WHEN we call Namespace() on the Grafana component
// THEN the call returns the namespace of that component
func TestNamespace(t *testing.T) {
	assert.Equal(t, "verrazzano-system", NewComponent().Namespace())
}

// TestShouldInstallBeforeUpgrade tests the ShouldInstallBeforeUpgrade function for the Grafana component
// GIVEN a component
// WHEN we call ShouldInstallBeforeUpgrade on the Grafana component
// THEN the call returns false
func TestShouldInstallBeforeUpgrade(t *testing.T) {
	assert.Equal(t, false, NewComponent().ShouldInstallBeforeUpgrade())
}

// TestGetDependencies tests the GetDependencies function for the Grafana component
// GIVEN a component
// WHEN we call GetDependencies on the Grafana component
// THEN the call returns a string array listing all the dependencies of the component
func TestGetDependencies(t *testing.T) {
	assert.Equal(t, []string{"verrazzano-network-policies", "verrazzano-monitoring-operator", "verrazzano-grafana-dashboards"}, NewComponent().GetDependencies())
}

// TestGetJSONName tests the GetJSONName function for the Grafana component
// GIVEN a component
// WHEN we call GetJSONName on the Grafana component
// THEN the call returns a string showing JSON name of the component
func TestGetJSONName(t *testing.T) {
	assert.Equal(t, ComponentJSONName, NewComponent().GetJSONName())
}

// TestGetMinVerrazzanoVersion tests the GetMinVerrazzanoVersion function for the Grafana component
// GIVEN a component
// WHEN we call GetMinVerrazzanoVersion on the Grafana component
// THEN the call returns a string showing the minimum verrazzano version for the component
func TestGetMinVerrazzanoVersion(t *testing.T) {
	assert.Equal(t, "1.0.0", NewComponent().GetMinVerrazzanoVersion())
}

// TestIsOperatorInstallSupported tests the IsOperatorInstallSupported function for the Grafana component
// GIVEN a component
// WHEN we call IsOperatorInstallSupported on the Grafana component
// THEN the call returns true
func TestIsOperatorInstallSupported(t *testing.T) {
	assert.Equal(t, true, NewComponent().IsOperatorInstallSupported())
}

// TestMonitorOverrides tests the MonitorOverrides function for the Grafana component
// GIVEN a component
// WHEN we call MonitorOverrides on the Grafana component
// THEN the call returns true
func TestMonitorOverrides(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	assert.Equal(t, true, NewComponent().MonitorOverrides(ctx))
}

// TestIsInstalled tests the IsInstalled function for the Grafana component
// GIVEN a component and a context
// WHEN we call IsInstalled on the Grafana component
// THEN the call returns true if grafana is installed and vice versa
func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			name: "Test isInstalled when Grafana is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			name:       "Test isInstalled when Grafana deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			isInstall, err := NewComponent().IsInstalled(ctx)
			assert.Equal(t, tt.expectTrue, isInstall)
			assert.NoError(t, err)
		})
	}
}

// TestIsAvailable tests the IsAvailable function for the Grafana component
// GIVEN a component and a context
// WHEN we call IsAvailable on the Grafana component
// THEN the call returns if the component is available and the reason for unavailability, if any
func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name              string
		component         grafanaComponent
		args              spi.ComponentContext
		expectedReason    string
		expectedAvailable vzapi.ComponentAvailability
	}{
		// GIVEN Default grafana component
		// WHEN  IsAvailable is called
		// THEN true is returned if component is available
		{
			"TestIsAvailable",
			grafanaComponent{},
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false),
			"waiting for deployment verrazzano-system/vmi-system-grafana to exist",
			vzapi.ComponentUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, available := tt.component.IsAvailable(tt.args)
			assert.Equal(t, reason, tt.expectedReason)
			assert.Equal(t, available, tt.expectedAvailable)
		})
	}

}

// TestIsReady tests the IsReady function for the Grafana component
// GIVEN a component and a context
// WHEN we call IsReady on the Grafana component
// THEN the call returns true if the component is ready and vice versa
func TestIsReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			name: "Test isReady when Grafana is successfully deployed and the admin secret exists",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
						Labels:    map[string]string{"app": "system-grafana"},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "system-grafana"},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"app":               "system-grafana",
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        grafanaDeployment + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.GrafanaSecret,
						Namespace: ComponentNamespace,
					},
					Data: map[string][]byte{},
				},
			).Build(),
			expectTrue: true,
		},
		{
			name: "Test isReady when Grafana is successfully deployed and the admin secret does not exist",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
		{
			name: "Test isReady when Grafana is deployed but there are no available replicas",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, NewComponent().IsReady(ctx))
			ctx = spi.NewFakeContext(tt.client, &vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Grafana: &vzapi.GrafanaComponent{Replicas: &replicas}}}}, nil, false)
			assert.Equal(t, true, NewComponent().IsReady(ctx))
		})
	}
}

// TestPostInstall tests the PostInstall function for the Grafana component
// GIVEN a component and a context
// WHEN we call PostInstall on the Grafana component
// THEN the call returns the post install conditions
func TestPostInstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, nil, nil, false)

	err := NewComponent().PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	vzComp := NewComponent()
	vzIngressNames := vzComp.(grafanaComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &nw.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	vzCertNames := vzComp.(grafanaComponent).GetCertificateNames(ctx)
	for _, certName := range vzCertNames {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = vzComp.(grafanaComponent).PostInstall(ctx)
	assert.Error(t, err)
	expectedErr := spi2.RetryableError{
		Source:    vzComp.Name(),
		Operation: "Check if certificates are ready",
	}
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

}

// TestIsOperatorUninstallSupported tests the IsOperatorUninstallSupported function for the Grafana component
// GIVEN a component
// WHEN we call IsOperatorUninstallSupported on the Grafana component
// THEN the call returns false
func TestIsOperatorUninstallSupported(t *testing.T) {
	assert.Equal(t, false, NewComponent().IsOperatorUninstallSupported())
}

// TestPreUninstall tests the PreUninstall function for the Grafana component
// GIVEN a component and a context
// WHEN we call PreUninstall on the Grafana component
// THEN the call returns nil
func TestPreUninstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	err := NewComponent().PreUninstall(ctx)
	assert.NoError(t, err)

}

// TestUninstall tests the Uninstall function for the Grafana component
// GIVEN a component and a context
// WHEN we call Uninstall on the Grafana component
// THEN the call returns nil
func TestUninstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	err := NewComponent().Uninstall(ctx)
	assert.NoError(t, err)

}

// TestPostUninstall tests the PostUninstall function for the Grafana component
// GIVEN a component and a context
// WHEN we call PostUninstall on the Grafana component
// THEN the call returns nil
func TestPostUninstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	err := NewComponent().PostUninstall(ctx)
	assert.NoError(t, err)

}

// TestPostUpgrade tests the PostUpgrade function for the Grafana component
// GIVEN a component and a context
// WHEN we call PostUpgrade on the Grafana component
// THEN the call returns post upgrade conditions
func TestPostUpgrade(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	err := NewComponent().PostUpgrade(ctx)
	assert.Error(t, err)

}

// TestReconcile tests the Reconcile function for the Grafana component
// GIVEN a component and a context
// WHEN we call Reconcile on the Grafana component
// THEN the call returns nil
func TestReconcile(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	err := NewComponent().Reconcile(ctx)
	assert.NoError(t, err)

}

// TestCheckExistingCNEGrafana tests the checkExistingCNEGrafana function for the Grafana component
// GIVEN a runtime object and enabled value of grafana as false
// WHEN we call checkExistingCNEGrafana
// THEN the call returns nil
func TestCheckExistingCNEGrafana(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{Enabled: &falseValue},
			},
		},
	}
	err := checkExistingCNEGrafana(vz)
	assert.NoError(t, err)

}

// TestGetOverrides tests the GetOverrides function for the Grafana component
// GIVEN a runtime object
// WHEN we call GetOverrides on the Grafana component
// THEN the call returns the overrides
func TestGetOverrides(t *testing.T) {
	tests := []struct {
		name     string
		actualCR vzapi.Verrazzano
	}{
		{
			name:     "Test1",
			actualCR: vzapi.Verrazzano{},
		},
	}
	ctx := spi.NewFakeContext(nil, &tests[0].actualCR, nil, false)
	assert.Equal(t, []vzapi.Overrides{}, NewComponent().GetOverrides(ctx.EffectiveCR()))

	assert.Equal(t, []v1beta1.Overrides{}, NewComponent().GetOverrides(nil))

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
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
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

	// make sure the VMI was created and the Grafana config is as expected
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "system", Namespace: globalconst.VerrazzanoSystemNamespace}, vmi)
	assert.NoError(t, err)
	assert.True(t, vmi.Spec.Grafana.Enabled)
	assert.Equal(t, vmi.Spec.Grafana.DashboardsConfigMap, "verrazzano-dashboard-provider")
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

// TestValidateUpdateV1beta1 tests the Grafana component ValidateUpdate function
func TestValidateUpdateV1beta1(t *testing.T) {
	// GIVEN an old VZ with Grafana enabled and a new VZ with Grafana disabled
	// WHEN we call the ValidateUpdate function
	// THEN the function returns an error
	oldVz := &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				Grafana: &v1beta1.GrafanaComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	newVz := &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				Grafana: &v1beta1.GrafanaComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	assert.Error(t, NewComponent().ValidateUpdateV1Beta1(oldVz, newVz))

	// GIVEN an old VZ with Grafana enabled and a new VZ with Grafana enabled
	// WHEN we call the ValidateUpdate function
	// THEN the function does not return an error
	newVz.Spec.Components.Grafana.Enabled = &trueValue
	assert.NoError(t, NewComponent().ValidateUpdateV1Beta1(oldVz, newVz))
}

func TestValidateInstall(t *testing.T) {
	svc := common.MkSvc(constants.IstioSystemNamespace, ComponentName)
	dep := common.MkDep(constants.IstioSystemNamespace, ComponentName)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Grafana: &vzapi.GrafanaComponent{},
			},
		},
	}
	common.RunValidateInstallTest(t, NewComponent,
		common.ValidateInstallTest{
			Name:      "NoExistingGrafana",
			WantErr:   "",
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(),
			Vz:        vz,
		},
		common.ValidateInstallTest{
			Name:      "ExistingDeployment",
			WantErr:   ComponentName,
			Appsv1Cli: common.MockGetAppsV1(dep),
			Corev1Cli: common.MockGetCoreV1(),
			Vz:        vz,
		},
		common.ValidateInstallTest{
			Name:      "ExistingService",
			WantErr:   ComponentName,
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(svc),
			Vz:        vz,
		})
}
