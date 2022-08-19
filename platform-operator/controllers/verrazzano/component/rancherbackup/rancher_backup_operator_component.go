// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

const (
	// ComponentName is the name of the component
	ComponentName = "rancher-backup"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.RancherBackupNamesSpace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "rancher-backup"
	// ChartDir is the name of the directory for third party helm charts
	ChartDir = "rancher-backup"
)

var (
	componentPrefix   = fmt.Sprintf("Component %s", ComponentName)
	rancherDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		rancherDeployment,
	}
)

type rancherBackupHelmComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return rancherBackupHelmComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
			ImagePullSecretKeyname:    imagePullSecretHelmKey,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "rancher-backup-override-static-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			Dependencies:              []string{rancher.ComponentName},
		},
	}
}

// IsEnabled returns true only if Rancher Backup component is explicitly enabled
// in the Verrazzano CR.
func (rb rancherBackupHelmComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.RancherBackup
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsInstalled returns true only if Rancher Backup is installed on the system
func (rb rancherBackupHelmComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	for _, nsn := range deployments {
		if err := ctx.Client().Get(context.TODO(), nsn, &appsv1.Deployment{}); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			// Unexpected error
			return false, err
		}
	}
	return true, nil
}

// validateRancherBackup checks scenarios in which the Verrazzano CR violates install verification
func (rb rancherBackupHelmComponent) validateRancherBackup(vz *vzapi.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.RancherBackup != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.RancherBackup.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (rb rancherBackupHelmComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.RancherBackup == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.RancherBackup.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.RancherBackup.MonitorChanges
	}
	return true
}

func (rb rancherBackupHelmComponent) PreInstall(ctx spi.ComponentContext) error {
	return ensureRancherBackupCrdInstall(ctx)
}

// IsReady checks if the RancherBackup objects are ready
func (rb rancherBackupHelmComponent) IsReady(ctx spi.ComponentContext) bool {
	return isRancherBackupOperatorReady(ctx)
}

func (rb rancherBackupHelmComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

// ValidateUpgrade verifies the install of the Verrazzano object
func (rb rancherBackupHelmComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return nil
}

func (rb rancherBackupHelmComponent) IsOperatorUninstallSupported() bool {
	return true
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (rb rancherBackupHelmComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if rb.IsEnabled(old) && !rb.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return rb.validateRancherBackup(new)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (rb rancherBackupHelmComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
}

// postUninstall processing for RancherBackup
func (rb rancherBackupHelmComponent) PostUninstall(context spi.ComponentContext) error {
	return postUninstall(context)
}
