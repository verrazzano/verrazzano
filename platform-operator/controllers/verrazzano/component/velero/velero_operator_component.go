// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	prometheusOperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
			ImagePullSecretKeyname:    imagePullSecretHelmKey,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "velero-override-static-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			Dependencies:              []string{networkpolicies.ComponentName, prometheusOperator.ComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DaemonsetNames:  daemonSets,
				DeploymentNames: deployments,
			},
		},
	}
}

// IsEnabled returns true only if Velero is explicitly enabled
// in the Verrazzano CR.
func (v veleroHelmComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsVeleroEnabled(effectiveCR)
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

// IsReady checks if the Velero objects are ready
func (v veleroHelmComponent) IsReady(ctx spi.ComponentContext) bool {
	return isVeleroOperatorReady(ctx)
}

func (v veleroHelmComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return v.HelmComponent.ValidateInstall(vz)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (v veleroHelmComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if v.IsEnabled(old) && !v.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return v.validateVelero(new)
}

// ValidateUpgrade verifies the install of the Verrazzano object
func (v veleroHelmComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return v.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (v veleroHelmComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	if v.IsEnabled(old) && !v.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	// Validate install overrides
	if new.Spec.Components.Velero != nil {
		if err := vzapi.ValidateInstallOverridesV1Beta1(new.Spec.Components.Velero.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// PostUninstall processing for Velero
func (v veleroHelmComponent) PostUninstall(context spi.ComponentContext) error {
	res := resource.Resource{
		Name:   ComponentNamespace,
		Client: context.Client(),
		Object: &corev1.Namespace{},
		Log:    context.Log(),
	}
	// Remove finalizers from the velero namespace to avoid hanging namespace deletion
	// and delete the namespace
	return res.RemoveFinalizersAndDelete()
}
