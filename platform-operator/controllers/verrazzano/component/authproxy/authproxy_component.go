// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"context"
	"fmt"
	"path/filepath"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
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

// PreInstall - actions to perform prior to installing this component
func (c authProxyComponent) PreInstall(ctx spi.ComponentContext) error {
	ctx.Log().Info("AuthProxy pre-install")

	// The AuthProxy helm chart was separated out of the Verrazzano helm chart in release 1.2.
	// During an upgrade from 1.1 to 1.2, there is a period of time when AuthProxy is being un-deployed
	// due to it being removed from the Verrazzano helm chart.  Wait for the undeploy to complete before
	// installing the AuthProxy helm chart.  This avoids Helm errors in the log of resources being
	// referenced by more than one chart.
	authProxySA := corev1.ServiceAccount{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}, &authProxySA)
	if err != nil && !errors.IsNotFound(err) {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	} else if err == nil {
		// Service account still exists, keep retrying
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: fmt.Errorf("")}
	}

	return nil
}
