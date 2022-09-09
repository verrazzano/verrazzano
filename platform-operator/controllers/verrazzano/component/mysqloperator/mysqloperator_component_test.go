// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
	overrides := comp.GetOverrides(vz).([]vzapi.Overrides)
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

	assert.True(t, isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
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
	assert.False(t, isReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
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

	assert.True(t, isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsInstalledFalse tests the isInstalled function
// GIVEN a call to isInstalled
//  WHEN the deployment object does not exist
//  THEN false is returned
func TestIsInstalledFalse(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()

	assert.False(t, isInstalled(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestValidateInstall tests the ValidateInstall function
// GIVEN a call to ValidateInstall
//  WHEN there is a valid MySQL Operator configuration
//  THEN the correct Helm overrides are returned
func TestValidateInstall(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		vz          *vzapi.Verrazzano
		expectError bool
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
			expectError: false,
		},
		{
			name: "MySQL Operator explicitly disabled, Keycloak enabled",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &trueValue}}}},
			expectError: true,
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
			expectError: false,
		},
		{
			name: "Keycloak enabled, MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			expectError: false,
		},
		{
			name: "Keycloak and MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{}}},
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().ValidateInstall(tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN the verrazzano-container-registry secret exists in the mysql-operator namespace
//  THEN the correct Helm overrides are returned
func TestAppendOverrides(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      constants.GlobalImagePullSecName,
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(secret).Build()

	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, nil, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
	assert.Equal(t, bom.KeyValue{Key: "image.pullSecrets.enabled", Value: "true"}, kvs[1])
}

// TestAppendOverridesNoSecret tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN the verrazzano-container-registry secret does not exist in the mysql-operator namespace
//  THEN the correct Helm overrides are returned
func TestAppendOverridesNoSecret(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, nil, nil, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 1)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
}
