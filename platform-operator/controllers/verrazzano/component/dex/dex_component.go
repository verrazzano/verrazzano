// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "dex"

// ComponentJSONName is the JSON name of the Dex component in CRD
const ComponentJSONName = "dex"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.DexNamespace

// DexComponent represents an Dex component
type DexComponent struct {
	helm.HelmComponent
}

var certificates = []types.NamespacedName{
	{Namespace: ComponentNamespace, Name: dexCertificateName},
}

// Verify that DexComponent implements Component
var _ spi.Component = DexComponent{}

// NewComponent returns a new Dex component
func NewComponent() spi.Component {
	return DexComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion2_0_0,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), helmValuesFile),
			Dependencies:              []string{networkpolicies.ComponentName, common.IstioComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName},
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendDexOverrides,
			Certificates:              certificates,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.DexIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// IsReady component check for Dex
func (c DexComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isDexReady(ctx)
	}
	return false
}

// isDexReady returns true if the availability objects that exist, have the minimum number of expected replicas
func (c DexComponent) isDexReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	// If a Dex component is enabled, ensure the deployment exists after the installation
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// IsEnabled returns whether Dex is enabled
func (c DexComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsDexEnabled(effectiveCR)
}

// PreInstall handles the pre-install operations for the Dex component
func (c DexComponent) PreInstall(ctx spi.ComponentContext) error {
	// Check Verrazzano Secret, create if it is not there
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}

	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return c.HelmComponent.PreInstall(ctx)
}

// PreUpgrade handles the pre-upgrade operations for the Dex component
func (c DexComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return c.HelmComponent.PreUpgrade(ctx)
}

func (c DexComponent) PostInstall(ctx spi.ComponentContext) error {
	// Update annotations on Dex Ingress
	err := updateDexIngress(ctx)
	if err != nil {
		return err
	}

	// Clean-up the overrides files created for static users and clients
	cleanTempFiles(ctx)

	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Dex component post-upgrade processing
func (c DexComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Dex component post-upgrade")

	// Clean-up the overrides files created for static users and clients
	cleanTempFiles(ctx)

	return c.HelmComponent.PostUpgrade(ctx)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c DexComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c DexComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	ctx.Log().Debugf("Deleting temp files using pattern: %v", tmpFileCleanPattern)
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}
