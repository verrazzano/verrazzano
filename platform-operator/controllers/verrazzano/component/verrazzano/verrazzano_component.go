// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ComponentName is the name of the component
	ComponentName = "verrazzano"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// vzImagePullSecretKeyName is the Helm key name for the VZ chart image pull secret
	vzImagePullSecretKeyName = "global.imagePullSecrets[0]"

	// ES secret keys
	esUsernameKey = "username"
	esPasswordKey = "password"
)

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "verrazzano"

var getControllerRuntimeClient = getClient

type verrazzanoComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return verrazzanoComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			ResolveNamespaceFunc:      resolveVerrazzanoNamespace,
			AppendOverridesFunc:       appendVerrazzanoOverrides,
			ImagePullSecretKeyname:    vzImagePullSecretKeyName,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			Dependencies:              []string{istio.ComponentName, nginx.ComponentName, certmanager.ComponentName, authproxy.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// PreInstall Verrazzano component pre-install processing; create and label required namespaces, copy any
// required secrets
func (c verrazzanoComponent) PreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		// Make sure the VMI CRD is installed since the Verrazzano component may create/update
		// a VMI CR
		if err := common.ApplyCRDYaml(ctx, config.GetHelmVMOChartsDir()); err != nil {
			return err
		}
	}
	// create or update  VMI secret
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	// create or update  backup secret
	if err := common.EnsureBackupSecret(ctx.Client()); err != nil {
		return err
	}
	ctx.Log().Debug("Verrazzano pre-install")
	if err := createAndLabelNamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespaces for Verrazzano: %v", err)
	}
	return nil
}

// Install Verrazzano component install processing
func (c verrazzanoComponent) Install(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.Install(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PreUpgrade Verrazzano component pre-upgrade processing
func (c verrazzanoComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		if err := common.ExportVMOHelmChart(ctx); err != nil {
			return err
		}
		if err := common.ApplyCRDYaml(ctx, config.GetHelmVMOChartsDir()); err != nil {
			return err
		}
	}
	return verrazzanoPreUpgrade(ctx)
}

// Upgrade Verrazzano component upgrade processing
func (c verrazzanoComponent) Upgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.Upgrade(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// IsReady component check
func (c verrazzanoComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isVerrazzanoReady(ctx)
	}
	return false
}

// IsInstalled component check
func (c verrazzanoComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return c.HelmComponent.IsInstalled(ctx)
}

// PostInstall - post-install, clean up temp files
func (c verrazzanoComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Verrazzano-post-upgrade processing
func (c verrazzanoComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano component post-upgrade")
	cleanTempFiles(ctx)
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		if err := common.ReassociateVMOResources(ctx); err != nil {
			return err
		}
	}
	if vzconfig.IsFluentdEnabled(ctx.EffectiveCR()) {
		if err := fluentd.ReassociateResources(ctx.Client()); err != nil {
			return err
		}
	}
	if vzconfig.IsJaegerOperatorEnabled(ctx.EffectiveCR()) {
		if err := jaegeroperator.ReassociateResources(ctx.Client()); err != nil {
			return err
		}
	}
	removeNodeExporterResources(ctx)
	return c.HelmComponent.PostUpgrade(ctx)
}

// IsEnabled verrazzano-specific enabled check for installation
func (c verrazzanoComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsVerrazzanoComponentEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c verrazzanoComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling active components
	if err := c.checkEnabled(old, new); err != nil {
		return err
	}
	// Reject any other edits except InstallArgs
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverrides(old, new, ComponentJSONName); err != nil {
		return err
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c verrazzanoComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c verrazzanoComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return c.HelmComponent.ValidateInstall(vz)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c verrazzanoComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return nil
}

func (c verrazzanoComponent) checkEnabled(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if vzconfig.IsConsoleEnabled(old) && !vzconfig.IsConsoleEnabled(new) {
		return fmt.Errorf("Disabling component console not allowed")
	}
	if vzconfig.IsPrometheusEnabled(old) && !vzconfig.IsPrometheusEnabled(new) {
		return fmt.Errorf("Disabling component prometheus not allowed")
	}
	return nil
}

// getClient returns a controller runtime client for the Verrazzano resource
func getClient() (client.Client, error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c verrazzanoComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Verrazzano != nil {
		if ctx.EffectiveCR().Spec.Components.Verrazzano.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Verrazzano.MonitorChanges
		}
		return true
	}
	return false
}
