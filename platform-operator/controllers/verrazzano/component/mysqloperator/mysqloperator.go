// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	config "github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

// AppendOverrides Build the set of MySQL operator overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	var secret corev1.Secret
	if err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: constants.GlobalImagePullSecName}, &secret); err != nil {
		if errors.IsNotFound(err) {
			// Global secret not found
			return kvs, nil
		}
		// we had an unexpected error
		return kvs, err
	}

	// We found the global secret, set the image.pullSecrets.enabled value to true
	kvs = append(kvs, bom.KeyValue{Key: "image.pullSecrets.enabled", Value: "true"})
	return kvs, nil
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
func (c mysqlOperatorComponent) validateMySQLOperator(vz *installv1beta1.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.MySQLOperator != nil {
		if err := vzapi.ValidateInstallOverridesV1Beta1(vz.Spec.Components.MySQLOperator.ValueOverrides); err != nil {
			return err
		}
	}
	// Must be enabled if Keycloak is enabled
	if config.IsKeycloakEnabled(vz) {
		if !c.IsEnabled(vz) {
			return fmt.Errorf("MySQLOperator must be enabled if Keycloak is enabled")
		}
	}
	return nil
}
