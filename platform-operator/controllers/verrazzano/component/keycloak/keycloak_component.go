// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package keycloak

import (
	"context"
	"fmt"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "keycloak-values.yaml"),
			Dependencies:            []string{istio.ComponentName},
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendKeycloakOverrides,
		},
	}
}

func (c KeycloakComponent) PreInstall(ctx spi.ComponentContext) error {
	// Check Verrazzano Secret. return error which will cause reque
	ctx.Log().Info("CDD Keycloak PreInstall Check Verrazzano Secret")
	secret := &corev1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: "verrazzano-system",
		Name:      "verrazzano",
	}, secret)
	if err != nil {
		ctx.Log().Errorf("loginKeycloak: Error retrieving Verrazzano password: %s", err)
		return err
	}
	ctx.Log().Info("CDD Keycloak PreInstall Verrazzano Secret Found")
	ctx.Log().Info("CDD Keycloak PreInstall Check MySQL Secret")
	// Check MySQL Secret. return error which will cause reque
	secret = &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: "keycloak",
		Name:      "mysql",
	}, secret)
	if err != nil {
		ctx.Log().Errorf("loginKeycloak: Error retrieving MySQL password: %s", err)
		return err
	}
	ctx.Log().Info("CDD Keycloak PreInstall MySQL Secret Found")

	// Create secret for the keycloakadmin user
	ctx.Log().Info("CDD Keycloak PreInstall Create Keycloak Secret")
	pw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	err = createOrUpdateAuthSecret(ctx, "keycloak", "keycloak-http", "keycloakadmin", pw)
	if err != nil {
		return err
	}
	ctx.Log().Info("CDD Keycloak PreInstall Keycloak Secret Successfully Created")
	return nil
}

func (c KeycloakComponent) PostInstall(ctx spi.ComponentContext) error {
	// Create secret for the verrazzano-prom-internal user
	prompw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	err = createOrUpdateAuthSecret(ctx, "verrazzano-system", "verrazzano-prom-internal", "verrazzano-prom-internal", prompw)
	if err != nil {
		return err
	}
	// Create secret for the verrazzano-es-internal user
	espw, err := vzpassword.GeneratePassword(15)
	if err != nil {
		return err
	}
	err = createOrUpdateAuthSecret(ctx, "verrazzano-system", "verrazzano-es-internal", "verrazzano-es-internal", espw)
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

func (c KeycloakComponent) IsEnabled(ctx spi.ComponentContext) bool {
	return isKeycloakEnabled(ctx)
}

func (c KeycloakComponent) IsReady(ctx spi.ComponentContext) bool {
	// TLS cert from Cert Manager should be in Ready state
	certName := fmt.Sprintf("%s-secret", getEnvironmentName(ctx.EffectiveCR().Spec.EnvironmentName))
	certificate := &certmanager.Certificate{}
	namespacedName := types.NamespacedName{Name: certName, Namespace: ComponentName}
	if err := ctx.Client().Get(context.TODO(), namespacedName, certificate); err != nil {
		ctx.Log().Infof("CDD Keycloak Failed to get Keycloak Certificate: %s", err)
		return false
	}
	if certificate.Status.Conditions == nil {
		ctx.Log().Infof("CDD Keycloak No Certificate Status conditions")
		return false
	}
	condition := certificate.Status.Conditions[0]
	return condition.Type == "Ready"
}

func isKeycloakEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Keycloak
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
