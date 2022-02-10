// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
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
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			Dependencies:            []string{verrazzano.ComponentName},
		},
	}
}

// IsEnabled authProxyComponent-specific enabled check for installation
func (c authProxyComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.AuthProxy
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady checks if the AuthProxy deployment is ready
func (c authProxyComponent) IsReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: ComponentName, Namespace: ComponentNamespace},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
