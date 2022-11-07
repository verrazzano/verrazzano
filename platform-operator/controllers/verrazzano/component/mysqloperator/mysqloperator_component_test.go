// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = scheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
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
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: ComponentNamespace}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("server error"))

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
			ctx:  spi.NewFakeContext(mockClient, &vzapi.Verrazzano{}, nil, false),
			err:  fmt.Errorf("Failed to create or update the mysql-operator namespace: server error"),
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
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("server error"))

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
			ctx:  spi.NewFakeContext(mockClient, &vzapi.Verrazzano{}, nil, false),
			err:  ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("server error")},
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

	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	mockClient.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("server error"))

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
			ctx:  spi.NewFakeContext(mockClient, nil, nil, false),
			err:  fmt.Errorf("server error"),
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
			name: "MySQL Operator explicitly enabled, Keycloak disabled",
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
			// MySQL Operator must be enabled if keycloak is enabled
			name: "MySQL Operator explicitly disabled, Keycloak enabled",
			vz: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						MySQLOperator: &installv1beta1.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &installv1beta1.KeycloakComponent{
							Enabled: &trueValue}}}},
			expectError: true,
		},
		{
			name: "MySQL Operator and Keycloak explicitly disabled",
			vz: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						MySQLOperator: &installv1beta1.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &installv1beta1.KeycloakComponent{
							Enabled: &falseValue}}}},
			expectError: false,
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
