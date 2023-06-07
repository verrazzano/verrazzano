// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "mysql"

// helmReleaseName is the name of the helm release
const helmReleaseName = ComponentName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.KeycloakNamespace

// DeploymentPersistentVolumeClaim is the name of a volume claim associated with a MySQL deployment
const DeploymentPersistentVolumeClaim = "mysql"

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "mysql"

// mysqlComponent represents an MySQL component
type mysqlComponent struct {
	helm.HelmComponent
}

// Verify that mysqlComponent implements Component
var _ spi.Component = mysqlComponent{}

// NewComponent returns a new MySQL component
func NewComponent() spi.Component {

	// Cannot include mysqloperatorcomponent because of import cycle
	const MySQLOperatorComponentName = "mysql-operator"

	return mysqlComponent{
		HelmComponent: helm.HelmComponent{
			ReleaseName:               helmReleaseName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "mysql-values.yaml"),
			AppendOverridesFunc:       appendMySQLOverrides,
			Dependencies:              []string{networkpolicies.ComponentName, istio.ComponentName, MySQLOperatorComponentName, fluentoperator.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				StatefulsetNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsReady calls MySQL isMySQLReady function
func (c mysqlComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return c.isMySQLReady(context)
	}
	return false
}

// IsEnabled mysql-specific enabled check for installation
// If keycloak is enabled, mysql is enabled; disabled otherwise
func (c mysqlComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsKeycloakEnabled(effectiveCR)
}

// PreInstall calls MySQL preInstall function
func (c mysqlComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstall(ctx, c.ChartNamespace); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

// PreUpgrade updates resources necessary for the MySQL Component upgrade
func (c mysqlComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := preUpgrade(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreUpgrade(ctx)
}

// PostInstall calls MySQL postInstall function
func (c mysqlComponent) PostInstall(ctx spi.ComponentContext) error {
	return postInstall(ctx)
}

// PostUpgrade creates/updates associated resources after this component is upgraded
func (c mysqlComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return postUpgrade(ctx)
}

// PostUninstall performs additional actions after the uninstall step
func (c mysqlComponent) PostUninstall(ctx spi.ComponentContext) error {
	return c.postUninstall(ctx)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Block all changes for now, particularly around storage changes

	// compare the VolumeSourceOverrides and reject if the type or size or storage class is different
	convertedOldVZ := v1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(old, &convertedOldVZ); err != nil {
		return err
	}
	oldSetting, err := doGenerateVolumeSourceOverrides(&convertedOldVZ, []bom.KeyValue{})
	if err != nil {
		return err
	}
	convertedNewVZ := v1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(new, &convertedNewVZ); err != nil {
		return err
	}
	newSetting, err := doGenerateVolumeSourceOverrides(&convertedNewVZ, []bom.KeyValue{})
	if err != nil {
		return err
	}
	// Reject any persistence-specific changes via the mysqlInstallArgs settings
	if err := validatePersistenceSpecificChanges(oldSetting, newSetting); err != nil {
		return err
	}
	// Reject any installArgs changes for now
	if err := common.CompareInstallArgs(c.getInstallArgs(old), c.getInstallArgs(new)); err != nil {
		return fmt.Errorf("Updates to mysqlInstallArgs not allowed for %s", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// compare the VolumeSourceOverrides and reject if the type or size or storage class is different
	oldSetting, err := doGenerateVolumeSourceOverrides(old, []bom.KeyValue{})
	if err != nil {
		return err
	}
	newSetting, err := doGenerateVolumeSourceOverrides(new, []bom.KeyValue{})
	if err != nil {
		return err
	}
	// Reject any persistence-specific changes via the mysqlInstallArgs settings
	if err := validatePersistenceSpecificChanges(oldSetting, newSetting); err != nil {
		return err
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// validatePersistenceSpecificChanges validates if there are any persistence related changes done via the install overrides
func validatePersistenceSpecificChanges(oldSetting, newSetting []bom.KeyValue) error {
	// Reject any persistence-specific changes via the mysqlInstallArgs settings
	if bom.FindKV(oldSetting, "datadirVolumeClaimTemplate.resources.requests.storage") != bom.FindKV(newSetting, "datadirVolumeClaimTemplate.resources.requests.storage") {
		return fmt.Errorf("Can not change persistence volume size in component: %s", ComponentJSONName)
	}
	if bom.FindKV(oldSetting, "datadirVolumeClaimTemplate.accessModes") != bom.FindKV(newSetting, "datadirVolumeClaimTemplate.accessModes") {
		return fmt.Errorf("Can not change persistence access modes in component: %s", ComponentJSONName)
	}
	if bom.FindKV(oldSetting, "datadirVolumeClaimTemplate.storageClassName") != bom.FindKV(newSetting, "datadirVolumeClaimTemplate.storageClassName") {
		return fmt.Errorf("Can not change storage class in component: %s", ComponentJSONName)
	}
	return nil
}

func (c mysqlComponent) getInstallArgs(vz *vzapi.Verrazzano) []vzapi.InstallArgs {
	if vz != nil && vz.Spec.Components.Keycloak != nil {
		return vz.Spec.Components.Keycloak.MySQL.MySQLInstallArgs
	}
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c mysqlComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Keycloak != nil {
		if ctx.EffectiveCR().Spec.Components.Keycloak.MySQL.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Keycloak.MySQL.MonitorChanges
		}
		return true
	}
	return false
}
