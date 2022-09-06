// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"fmt"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	config "github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

// getOverrides gets the install overrides
func getOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.MySQLOperator != nil {
			return effectiveCR.Spec.Components.MySQLOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.MySQLOperator != nil {
			return effectiveCR.Spec.Components.MySQLOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// isReady - component specific checks for being ready
func isReady(ctx spi.ComponentContext) bool {
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), getDeploymentList(), 1, getPrefix(ctx))
}

// isInstalled checks that the deployment exists
func isInstalled(ctx spi.ComponentContext) bool {
	return status.DoDeploymentsExist(ctx.Log(), ctx.Client(), getDeploymentList(), 1, getPrefix(ctx))
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}

func getDeploymentList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
}

// validateMySQLOperator checks scenarios in which the Verrazzano CR violates install verification
// MySQLOperator must be enabled if Keycloak is enabled
func (c mysqlOperatorComponent) validateMySQLOperator(object runtime.Object) error {
	// Validate install overrides
	if vz, ok := object.(*vzapi.Verrazzano); ok {
		if vz.Spec.Components.MySQLOperator != nil {
			if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.MySQLOperator.ValueOverrides); err != nil {
				return err
			}
		}
	} else if vz, ok := object.(*installv1beta1.Verrazzano); ok {
		if vz.Spec.Components.MySQLOperator != nil {
			if err := vzapi.ValidateInstallOverridesV1Beta1(vz.Spec.Components.MySQLOperator.ValueOverrides); err != nil {
				return err
			}
		}
	}
	// Must be enabled if Keycloak is enabled
	if config.IsKeycloakEnabled(object) {
		if !c.IsEnabled(object) {
			return fmt.Errorf("MySQLOperator must be enabled if Keycloak is enabled")
		}
	}
	return nil
}
