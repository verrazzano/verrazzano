// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-authproxy"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

type authProxyComponent struct {
	helm.HelmComponent
}

// Verify that AuthProxyComponent implements Component
var _ spi.Component = authProxyComponent{}

// NewComponent returns a new authProxyComponent component
func NewComponent() spi.Component {
	return authProxyComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			Dependencies:            []string{verrazzano.ComponentName},
		},
	}
}

// IsEnabled authProxyComponent-specific enabled check for installation
func (c authProxyComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.AuthProxy
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c authProxyComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("can not disable previously enabled authProxy")
	}
	return nil
}

// IsReady component check
func (c authProxyComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isAuthProxyReady(ctx)
	}
	return false
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (c authProxyComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	ingressNames := []types.NamespacedName{
		{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.VzConsoleIngress,
		},
	}
	return ingressNames
}

// PreInstall - actions to perform prior to installing this component
func (c authProxyComponent) PreInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debug("AuthProxy pre-install")

	// Temporary work around for installer bug of calling pre-install after a component is installed
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		ctx.Log().Oncef("Component %s already installed, skipping PreInstall checks", ComponentName)
		return nil
	}

	// The AuthProxy helm chart was separated out of the Verrazzano helm chart in release 1.2.
	// During an upgrade from 1.1 to 1.2, there is a period of time when AuthProxy is being un-deployed
	// due to it being removed from the Verrazzano helm chart.  Wait for the undeploy to complete before
	// installing the AuthProxy helm chart.  This avoids Helm errors in the log of resources being
	// referenced by more than one chart.
	authProxySA := corev1.ServiceAccount{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, &authProxySA)
	if (err == nil) || (err != nil && !errors.IsNotFound(err)) {
		ctx.Log().Progressf("Component %s is waiting for pre-install conditions to be met", ComponentName)
		return fmt.Errorf("Waiting for ServiceAccount %s to not exist", ComponentName)
	}
	// Continuing because expected condition of "ServiceAccount for AuthProxy not found" met

	return nil
}
