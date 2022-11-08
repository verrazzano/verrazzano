// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"context"
	"fmt"
	"testing"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = scheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

const (
	serverErr = "server-error"
	testName1 = "MySQL Operator explicitly enabled, Keycloak disabled"
	testName2 = "MySQL Operator and Keycloak explicitly disabled"
	testName3 = "MySQL Operator explicitly disabled, Keycloak enabled"
)

type erroringFakeClient struct {
	client.Client
}

func (e *erroringFakeClient) Get(_ context.Context, _ types.NamespacedName, _ client.Object) error {
	return fmt.Errorf(serverErr)
}

func (e *erroringFakeClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return fmt.Errorf(serverErr)
}

// TestIsEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN varying the enabled states of keycloak and MySQL Operator
// THEN check for the expected response
func TestIsEnabled(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name string
		vz   *vzapi.Verrazzano
		want bool
	}{
		{
			name: testName1,
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
			name: testName2,
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

// TestValidateInstall tests the ValidateInstall function
// GIVEN a call to ValidateInstall
// WHEN there is a valid MySQL Operator configuration
// THEN the correct Helm overrides are returned
func TestValidateInstall(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		vz          *vzapi.Verrazzano
		expectError bool
	}{
		{
			name: testName1,
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
			name: testName2,
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
			name: testName3,
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
		{
			name:        "Nil vz resource",
			vz:          nil,
			expectError: true,
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

// TestPreInstall tests the PreInstall function of mysqloperator
// GIVEN a call to PreInstall with different vz CR
// THEN return the expected error
func TestPreInstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	enabled := true
	erroringClient := &erroringFakeClient{fakeClient}

	tests := []struct {
		name string
		ctx  spi.ComponentContext
		err  error
	}{
		{
			name: "PreInstallNoError",
			ctx:  spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false),
			err:  nil,
		},
		{
			name: "PreInstallWithIstioEnabled",
			ctx: spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled:          &enabled,
							InjectionEnabled: &enabled,
						},
					},
				},
			}, nil, false),
			err: nil,
		},
		{
			name: "PreInstallServerError",
			ctx:  spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, nil, false),
			err:  fmt.Errorf("Failed to create or update the mysql-operator namespace: %s", serverErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().PreInstall(tt.ctx)
			assert.Equal(t, tt.err, err)
		})
	}
}

// TestPreUpgrade tests the PreUpgrade function of mysqloperator
// GIVEN a call to PreUpgrade
// THEN return the expected error
func TestPreUpgrade(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	erroringClient := &erroringFakeClient{fakeClient}

	tests := []struct {
		name string
		ctx  spi.ComponentContext
		err  error
	}{
		{
			// If the mysql-operator deployment is found, update and return no error
			name: "PreUpgradeNoError",
			ctx:  spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false),
			err:  nil,
		},
		{
			// If the mysql-operator deployment is not found for any reason, return a retryable error
			name: "PreUpgradeServerError",
			ctx:  spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, nil, false),
			err:  ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf(serverErr)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().PreUpgrade(tt.ctx)
			assert.Equal(t, tt.err, err)
		})
	}
}

// TestPreUninstall tests the PreUninstall function of mysqloperator
// GIVEN a call to PreUninstall
// THEN return a retryable error
// WHEN mysql pods are still present in the keycloak ns
func TestPreUninstall(t *testing.T) {
	label := make(map[string]string)
	label["mysql.oracle.com/cluster"] = "mysql"

	fakeClient := fake.NewClientBuilder().WithLists(
		&corev1.PodList{
			TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: constants.KeycloakNamespace,
						Labels:    label,
					},
				},
			},
		},
	).Build()

	erroringClient := &erroringFakeClient{fakeClient}

	tests := []struct {
		name string
		ctx  spi.ComponentContext
		err  error
	}{
		{
			name: "PreUninstallNoError",
			ctx:  spi.NewFakeContext(fake.NewClientBuilder().Build(), nil, nil, false),
			err:  nil,
		},
		{
			name: "PreUninstallRetryableError",
			ctx:  spi.NewFakeContext(fakeClient, nil, nil, false),
			err:  ctrlerrors.RetryableError{Source: ComponentName},
		},
		{
			name: "PreUninstallServerError",
			ctx:  spi.NewFakeContext(erroringClient, nil, nil, false),
			err:  fmt.Errorf(serverErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().PreUninstall(tt.ctx)
			assert.Equal(t, tt.err, err)
		})
	}
}

// TestValidateUpdate tests ValidateUpdate function
// GIVEN an old and a new vz resource
// THEN return an error
// WHEN mysqloperator is getting disabled during update
func TestValidateUpdate(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		oldvz       *vzapi.Verrazzano
		newvz       *vzapi.Verrazzano
		expectError bool
	}{
		{
			name:        "MySQL Operator enabled in both new and old vz",
			oldvz:       getVZWithMySQLOperatorComp(&trueValue),
			newvz:       getVZWithMySQLOperatorComp(&trueValue),
			expectError: false,
		},
		{
			name:        "MySQL Operator enabled in old vz and disabled in new vz",
			oldvz:       getVZWithMySQLOperatorComp(&trueValue),
			newvz:       getVZWithMySQLOperatorComp(&falseValue),
			expectError: true,
		},
		{
			name:        "MySQL Operator component nil in both new and old vz",
			oldvz:       getVZWithMySQLOperatorComp(nil),
			newvz:       getVZWithMySQLOperatorComp(nil),
			expectError: false,
		},
		{
			name:        "MySQL Operator component nil in old and disabled in new",
			oldvz:       getVZWithMySQLOperatorComp(nil),
			newvz:       getVZWithMySQLOperatorComp(&falseValue),
			expectError: true,
		},
		{
			name:        "New vz is nil",
			oldvz:       getVZWithMySQLOperatorComp(&trueValue),
			newvz:       nil,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().ValidateUpdate(tt.oldvz, tt.newvz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateInstallV1Beta1 tests ValidateInstallV1Beta1 function
// GIVEN a v1beta1 vz resource
// THEN return the expected error
func TestValidateInstallV1Beta1(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		vz          *installv1beta1.Verrazzano
		expectError bool
	}{
		{
			name: testName1,
			vz: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						MySQLOperator: &installv1beta1.MySQLOperatorComponent{
							Enabled: &trueValue},
						Keycloak: &installv1beta1.KeycloakComponent{
							Enabled: &falseValue}}}},
			expectError: false,
		},
		{
			name: testName2,
			vz: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						MySQLOperator: &installv1beta1.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &installv1beta1.KeycloakComponent{
							Enabled: &falseValue}}}},
			expectError: false,
		},
		{
			// MySQL Operator must be enabled if keycloak is enabled
			name: testName3,
			vz: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						MySQLOperator: &installv1beta1.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &installv1beta1.KeycloakComponent{
							Enabled: &trueValue}}}},
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().ValidateInstallV1Beta1(tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdate tests ValidateUpdate function
// GIVEN an old and a new v1beta1 vz resource
// THEN return an error
// WHEN mysqloperator is getting disabled during update
func TestValidateUpdateV1Beta1(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		oldvz       *installv1beta1.Verrazzano
		newvz       *installv1beta1.Verrazzano
		expectError bool
	}{
		{
			name:        "MySQL Operator enabled in both new and old vz",
			oldvz:       getv1beta1VZWithMySQLOperatorComp(&trueValue),
			newvz:       getv1beta1VZWithMySQLOperatorComp(&trueValue),
			expectError: false,
		},
		{
			name:        "MySQL Operator enabled in old vz and disabled in new vz",
			oldvz:       getv1beta1VZWithMySQLOperatorComp(&trueValue),
			newvz:       getv1beta1VZWithMySQLOperatorComp(&falseValue),
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewComponent().ValidateUpdateV1Beta1(tt.oldvz, tt.newvz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// getVZWithMySQLOperatorComp return a v1alpha1 vz resource with mysql-operator enabled, disabled or nil
func getVZWithMySQLOperatorComp(enabled *bool) *vzapi.Verrazzano {
	if enabled == nil {
		return &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					MySQLOperator: nil,
				}}}
	}
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				MySQLOperator: &vzapi.MySQLOperatorComponent{
					Enabled: enabled,
				},
			}}}
}

// getv1beta1VZWithMySQLOperatorComp return a v1beta1 vz resource with mysql-operator enabled or disabled
func getv1beta1VZWithMySQLOperatorComp(enabled *bool) *installv1beta1.Verrazzano {
	return &installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				MySQLOperator: &installv1beta1.MySQLOperatorComponent{
					Enabled: enabled,
				},
			}}}
}
