// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"context"
	"fmt"
	"path/filepath"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "keycloak"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.KeycloakNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "keycloak"

// KeycloakComponent represents an Keycloak component
type KeycloakComponent struct {
	helm.HelmComponent
}

// Verify that KeycloakComponent implements Component
var _ spi.Component = KeycloakComponent{}

var certificates = []types.NamespacedName{
	{Namespace: ComponentNamespace, Name: keycloakCertificateName},
}

// NewComponent returns a new Keycloak component
func NewComponent() spi.Component {
	return KeycloakComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "keycloak-values.yaml"),
			Dependencies:              []string{istio.ComponentName, nginx.ComponentName, certmanager.ComponentName},
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendKeycloakOverrides,
			Certificates:              certificates,
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.KeycloakIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// Reconcile - the only condition currently being handled by this function is to restore
// the Keycloak configuration when the MySQL pod gets restarted and ephemeral storage is being used.
func (c KeycloakComponent) Reconcile(ctx spi.ComponentContext) error {
	// If the Keycloak component is ready, confirm the configuration is working.
	// If ephemeral storage is being used, the Keycloak configuration will be rebuilt if needed.
	if isKeycloakReady(ctx) {
		ctx.Log().Debugf("Component %s calling configureKeycloakRealms from Reconcile", ComponentName)
		return configureKeycloakRealms(ctx)
	}
	return fmt.Errorf("Component %s not ready yet to check configuration", ComponentName)
}

func (c KeycloakComponent) PreInstall(ctx spi.ComponentContext) error {
	// Check Verrazzano Secret. return error which will cause requeue
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
	// Check MySQL Secret. return error which will cause requeue
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

	// Update annotations on Keycloak Ingress
	err = updateKeycloakIngress(ctx)
	if err != nil {
		return err
	}

	// Update the Prometheus annotations to include the Keycloak service as an outbound IP address
	if promoperator.NewComponent().IsEnabled(ctx.EffectiveCR()) {
		err = common.UpdatePrometheusAnnotations(ctx, promoperator.ComponentNamespace, promoperator.ComponentName)
		if err != nil {
			return err
		}
	}

	return c.HelmComponent.PostInstall(ctx)
}

// PreUpgrade - component level processing for pre-upgrade
func (c KeycloakComponent) PreUpgrade(ctx spi.ComponentContext) error {
	// Determine if additional processing is required for the upgrade of the StatefulSet
	return upgradeStatefulSet(ctx)
}

// PostUpgrade Keycloak-post-upgrade processing, create or update the Kiali ingress
func (c KeycloakComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}

	// Update the Prometheus annotations to include the Keycloak service as an outbound IP address
	if promoperator.NewComponent().IsEnabled(ctx.EffectiveCR()) {
		err := common.UpdatePrometheusAnnotations(ctx, promoperator.ComponentNamespace, promoperator.ComponentName)
		if err != nil {
			return err
		}
	}

	return configureKeycloakRealms(ctx)
}

// IsEnabled Keycloak-specific enabled check for installation
func (c KeycloakComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsKeycloakEnabled(effectiveCR)
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
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	// Reject any other edits for now
	if err := common.CompareInstallArgs(c.getInstallArgs(old), c.getInstallArgs(new)); err != nil {
		return fmt.Errorf("Updates to InstallArgs not allowed for %s", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c KeycloakComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

func (c KeycloakComponent) getInstallArgs(vz *vzapi.Verrazzano) []vzapi.InstallArgs {
	if vz != nil && vz.Spec.Components.Keycloak != nil {
		return vz.Spec.Components.Keycloak.KeycloakInstallArgs
	}
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c KeycloakComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Keycloak != nil {
		if ctx.EffectiveCR().Spec.Components.Keycloak.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Keycloak.MonitorChanges
		}
		return true
	}
	return false
}
