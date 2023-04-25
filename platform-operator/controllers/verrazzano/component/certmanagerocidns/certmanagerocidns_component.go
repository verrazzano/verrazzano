// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerocidns

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager-ocidns"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "cert-manager"

// certManagerOciDnsComponent represents an CertManager component
type certManagerOciDNSComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerOciDNSComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerOciDNSComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "cert-manager-webhook-oci"),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "cert-manager-ocidns-values.yaml"),
			//AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion: constants.VerrazzanoVersion1_0_0,
			Dependencies:         []string{networkpolicies.ComponentName, certmanager.ComponentName},
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerOciDNSComponent) IsEnabled(effectiveCR runtime.Object) bool {
	logger := vzlog.DefaultLogger()
	err := common.CertManagerExistsInCluster(logger)
	if err != nil {
		logger.ErrorfThrottled("Unexpected error checking for CertManager in cluster: %v", err)
		return false
	}
	return vzcr.IsExternalDNSEnabled(effectiveCR)
}

// IsReady component check
func (c certManagerOciDNSComponent) IsReady(ctx spi.ComponentContext) bool {
	if !isCertManagerEnabled(ctx) {
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return isCertManagerOciDNSReady(ctx)
	}
	return false
}

// IsCertManagerEnabled returns true if the cert-manager is enabled, which is the default
func isCertManagerEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
