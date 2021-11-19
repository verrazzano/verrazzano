// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package keycloak

import (
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "keycloak"

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
			ChartNamespace:          ComponentName,
			IgnoreNamespaceOverride: true,
			//  Check on Image Pull Pull Key
			ValuesFile:          filepath.Join(config.GetHelmOverridesDir(), "keycloak-values.yaml"),
			Dependencies:        []string{istio.ComponentName},
			AppendOverridesFunc: AppendKeycloakOverrides,
		},
	}
}

func (c KeycloakComponent) PreInstall(ctx spi.ComponentContext) error {
	// Check Verrazzano Secret. return error which will cause reque
	_, err := pkg.GetSecret("verrazzano-system", "verrazzano")
	if err != nil {
		return err
	}
	// Check MySQL Secret. return error which will cause reque
	_, err = pkg.GetSecret("keycloak", "mysql")
	if err != nil {
		return err
	}

	// Create secret for the keycloakadmin user
	pw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	pkg.CreatePasswordSecret("keycloak", "keycloak-http", pw, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c KeycloakComponent) PostInstall(ctx spi.ComponentContext) error {
	// Create secret for the verrazzano-prom-internal user
	prompw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	pkg.CreatePasswordSecret("verrazzano-system", "verrazzano-prom-internal", prompw, nil)
	if err != nil {
		return err
	}

	// Create secret for the verrazzano-es-internal user
	espw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	pkg.CreatePasswordSecret("verrazzano-system", "verrazzano-es-internal", espw, nil)
	if err != nil {
		return err
	}

	// Create the verrazzano-system realm and populate it with users, groups, clients, etc.
	err = configureKeycloakRealms(ctx, prompw, espw)
	if err != nil {
		return err
	}

	return nil
}

// PostUpgrade Keycloak-post-upgrade processing, create or update the Kiali ingress
func (c KeycloakComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Infof("Keycloak post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return updateKeycloakUris(ctx)
}

func isKeycloakEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Keycloak
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

func (c KeycloakComponent) IsReady(ctx spi.ComponentContext) bool {
	//   Wait for TLS cert from Cert Manager to go into a ready state   USE isREady
	certName := "cert/" + getEnvironmentName(ctx.EffectiveCR().Spec.EnvironmentName) + "-secret"
	cmd := execCommand("kubectl", "wait", certName, "-n", "keycloak", "--for=condition=Ready", "--timeout=5s")
	out, err := cmd.Output()
	if err != nil {
		ctx.Log().Errorf("Keycloak component isReady returned false, TLS Cert not in Ready state: Error  = %s", out)
		return false
	}
	return true
}
