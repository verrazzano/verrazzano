// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

const (
	// ComponentName is the name of the component
	ComponentName = "velero"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VeleroNameSpace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "velero"
	// ChartDir is the name of the directory for third party helm charts
	ChartDir = "velero"
)

var (
	componentPrefix  = fmt.Sprintf("Component %s", ComponentName)
	veleroDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	resticDaemonset = types.NamespacedName{
		Name:      constants.ResticDaemonSetName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		veleroDeployment,
	}

	daemonSets = []types.NamespacedName{
		resticDaemonset,
	}
)

type veleroHelmComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return veleroHelmComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_4_0,
			ImagePullSecretKeyname:  "image.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "velero-override-static-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			GetInstallOverridesFunc: GetOverrides,
			Dependencies:            []string{},
		},
	}
}

// IsEnabled returns true only if Velero is explicitly enabled
// in the Verrazzano CR.
func (v veleroHelmComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Velero
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsInstalled returns true only if Velero is installed on the system
func (v veleroHelmComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
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

// validateVelero checks scenarios in which the Verrazzano CR violates install verification
func (v veleroHelmComponent) validateVelero(vz *vzapi.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.Velero != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.Velero.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (v veleroHelmComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Velero == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.Velero.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.Velero.MonitorChanges
	}
	return true
}

func (v veleroHelmComponent) PreInstall(ctx spi.ComponentContext) error {
	return ensureVeleroNamespace(ctx)
}

// IsReady checks if the Jaeger Operator deployment is ready
func (v veleroHelmComponent) IsReady(ctx spi.ComponentContext) bool {
	return isVeleroOperatorReady(ctx)
}

func (v veleroHelmComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (v veleroHelmComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if v.IsEnabled(old) && !v.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return v.validateVelero(new)
}
