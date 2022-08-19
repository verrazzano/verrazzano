// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"testing"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestGetInstallOverrides tests the GetInstallOverrides function
// GIVEN a call to GetInstallOverrides
//  WHEN there is a valid MySQL Operator configuration
//  THEN the correct Helm overrides are returned
func TestGetInstallOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				MySQLOperator: &vzapi.MySQLOperatorComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: []byte("{\"key1\": \"value1\"}")},
							},
						},
					},
				},
			},
		},
	}

	comp := NewComponent()
	overrides := comp.GetOverrides(vz)
	assert.Equal(t, []byte("{\"key1\": \"value1\"}"), overrides[0].Values.Raw)
}

// TestIsEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN varying the enabled states of keycloak and MySQL Operator
//  THEN check for the expected response
func TestIsEnabled(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name string
		vz   *vzapi.Verrazzano
		want bool
	}{
		{
			name: "MySQL Operator explicitly enabled, Keycloak disabled",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{
							Enabled: &trueValue},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: true,
		},
		{
			name: "MySQL Operator and Keycloak explicitly disabled",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: false,
		},
		{
			name: "Keycloak enabled, MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: true,
		},
		{
			name: "Keycloak and MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().IsEnabled(tt.vz)
			assert.Equal(t, tt.want, c)
		})
	}
}

// TestIsMySQLOperatorReady tests the isReady function
// GIVEN a call to isReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsMySQLOperatorReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               ComponentName,
				},
			},
		},
	).Build()

	assert.True(t, isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsMySQLOperatorNotReady tests the isReady function
// GIVEN a call to isReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsMySQLOperatorNotReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentNamespace,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	assert.False(t, isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsInstalled tests the isInstalled function
// GIVEN a call to isInstalled
//  WHEN the deployment object exists
//  THEN true is returned
func TestIsInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
		},
	).Build()

	assert.True(t, isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsInstalledFalse tests the isInstalled function
// GIVEN a call to isInstalled
//  WHEN the deployment object does not exist
//  THEN false is returned
func TestIsInstalledFalse(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()

	assert.False(t, isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}
