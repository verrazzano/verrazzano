// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package keycloak

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"path/filepath"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "keycloak"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.KeycloakNamespace

// KeycloakComponent represents an Keycloak component
type KeycloakComponent struct {
	helm.HelmComponent
}

// Verify that KeycloakComponent implements Component
var _ spi.Component = KeycloakComponent{}

// NewComponent returns a new Keycloak component
func NewComponent() spi.Component {
	return KeycloakComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			//  Check on Image Pull Key
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "keycloak-values.yaml"),
			Dependencies:            []string{istio.ComponentName},
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendKeycloakOverrides,
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.KeycloakIngress,
				},
			},
		},
	}
}

func (c KeycloakComponent) PreInstall(ctx spi.ComponentContext) error {
	// Check Verrazzano Secret. return error which will cause reque
	secret := &corev1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.Verrazzano,
	}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Log().Progressf("Component Keycloak waiting for the Verrazzano password %s/%s to exist",
				constants.VerrazzanoSystemNamespace, constants.Verrazzano)
			return ctrlerrors.RetryableError{Source: ComponentName}
		}
		ctx.Log().Errorf("Component Keycloak failed to get the Verrazzano password %s/%s: %v",
			constants.VerrazzanoSystemNamespace, constants.Verrazzano, err)
		return err
	}
	// Check MySQL Secret. return error which will cause reque
	secret = &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: ComponentNamespace,
		Name:      mysql.ComponentName,
	}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			ctx.Log().Progressf("Component Keycloak waiting for the MySql password %s/%s to exist", ComponentNamespace, mysql.ComponentName)
			return ctrlerrors.RetryableError{Source: ComponentName}
		}
		ctx.Log().Errorf("Component Keycloak failed to get the MySQL password %s/%s: %v", ComponentNamespace, mysql.ComponentName, err)
		return err
	}

	// Create secret for the keycloakadmin user if it doesn't exist
	err = createAuthSecret(ctx, ComponentNamespace, "keycloak-http", "keycloakadmin")
	if err != nil {
		return err
	}

	return nil
}

func (c KeycloakComponent) PostInstall(ctx spi.ComponentContext) error {
	// Create secret for the verrazzano-prom-internal user
	err := createAuthSecret(ctx, constants.VerrazzanoSystemNamespace, "verrazzano-prom-internal", "verrazzano-prom-internal")
	if err != nil {
		return err
	}

	// Create secret for the verrazzano-es-internal user
	err = createAuthSecret(ctx, constants.VerrazzanoSystemNamespace, "verrazzano-es-internal", "verrazzano-es-internal")
	if err != nil {
		return err
	}
	// Create the verrazzano-system realm and populate it with users, groups, clients, etc.
	err = configureKeycloakRealms(ctx)
	if err != nil {
		return err
	}

	// If OCI DNS, update annotation on Keycloak Ingress
	if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
		err := updateKeycloakIngress(ctx)
		if err != nil {
			return err
		}
	}

	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Keycloak-post-upgrade processing, create or update the Kiali ingress
func (c KeycloakComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return updateKeycloakUris(ctx)
}

// IsEnabled Keycloak-specific enabled check for installation
func (c KeycloakComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Keycloak
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady component check
func (c KeycloakComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isKeycloakReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c KeycloakComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("can not disable previously enabled keycloak")
	}
	return nil
}
