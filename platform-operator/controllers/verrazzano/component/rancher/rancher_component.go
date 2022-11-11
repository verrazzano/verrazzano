// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"strconv"
)

// ComponentName is the name of the component
const ComponentName = common.RancherName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = common.CattleSystem

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "rancher"

type rancherComponent struct {
	helm.HelmComponent
}

var certificates = []types.NamespacedName{
	{Name: "tls-rancher-ingress", Namespace: ComponentNamespace},
}

func NewComponent() spi.Component {
	return rancherComponent{
		HelmComponent: helm.HelmComponent{
			ReleaseName:             common.RancherName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), common.RancherName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "rancher-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			Certificates:            certificates,
			Dependencies:            []string{nginx.ComponentName, certmanager.ComponentName},
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.RancherIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// AppendOverrides set the Rancher overrides for Helm
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	rancherHostName, err := getRancherHostname(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   "hostname",
		Value: rancherHostName,
	})
	// Always set useBundledChart=true
	kvs = append(kvs, bom.KeyValue{
		Key:   useBundledSystemChartKey,
		Value: useBundledSystemChartValue,
	})
	kvs = appendRegistryOverrides(kvs)
	return appendCAOverrides(kvs, ctx)
}

// appendRegistryOverrides appends overrides if a custom registry is being used
func appendRegistryOverrides(kvs []bom.KeyValue) []bom.KeyValue {
	// If using external registry, add registry overrides to Rancher
	registry := os.Getenv(constants.RegistryOverrideEnvVar)
	if registry != "" {
		imageRepo := os.Getenv(constants.ImageRepoOverrideEnvVar)
		var rancherRegistry string
		if imageRepo == "" {
			rancherRegistry = registry
		} else {
			rancherRegistry = fmt.Sprintf("%s/%s", registry, imageRepo)
		}
		kvs = append(kvs, bom.KeyValue{
			Key:   systemDefaultRegistryKey,
			Value: rancherRegistry,
		})
	}
	return kvs
}

// appendCAOverrides sets overrides for CA Issuers, ACME or CA.
func appendCAOverrides(kvs []bom.KeyValue, ctx spi.ComponentContext) ([]bom.KeyValue, error) {
	cm := ctx.EffectiveCR().Spec.Components.CertManager
	if cm == nil {
		return kvs, errors.New("certManager component not found in effective cr")
	}

	// Configure CA Issuer KVs
	if (cm.Certificate.Acme != vzapi.Acme{}) {
		kvs = append(kvs,
			bom.KeyValue{
				Key:   letsEncryptIngressClassKey,
				Value: common.RancherName,
			}, bom.KeyValue{
				Key:   letsEncryptEmailKey,
				Value: cm.Certificate.Acme.EmailAddress,
			}, bom.KeyValue{
				Key:   letsEncryptEnvironmentKey,
				Value: cm.Certificate.Acme.Environment,
			}, bom.KeyValue{
				Key:   ingressTLSSourceKey,
				Value: letsEncryptTLSSource,
			}, bom.KeyValue{
				Key:   additionalTrustedCAsKey,
				Value: strconv.FormatBool(useAdditionalCAs(cm.Certificate.Acme)),
			})
	} else { // Certificate issuer type is CA
		kvs = append(kvs, bom.KeyValue{
			Key:   ingressTLSSourceKey,
			Value: caTLSSource,
		})
		if isUsingDefaultCACertificate(cm) {
			kvs = append(kvs, bom.KeyValue{
				Key:   privateCAKey,
				Value: privateCAValue,
			})
		}
	}

	return kvs, nil
}

// IsEnabled Rancher is always enabled on admin clusters,
// and is not enabled by default on managed clusters
func (r rancherComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Rancher
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (r rancherComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Block all changes for now, particularly around storage changes
	if r.IsEnabled(old) && !r.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return r.HelmComponent.ValidateUpdate(old, new)
}

// PreInstall
/* Sets up the environment for Rancher
- Create the Rancher namespace if it is not present (cattle-namespace)
- note: VZ-5241 the rancher-operator-namespace is no longer used in 2.6.3
- Copy TLS certificates for Rancher if using the default Verrazzano CA
- Create additional LetsEncrypt TLS certificates for Rancher if using LE
*/
func (r rancherComponent) PreInstall(ctx spi.ComponentContext) error {
	vz := ctx.EffectiveCR()
	c := ctx.Client()
	log := ctx.Log()
	if err := createCattleSystemNamespace(log, c); err != nil {
		return err
	}
	if err := copyDefaultCACertificate(log, c, vz); err != nil {
		return err
	}
	return nil
}

// PreUpgrade
/* Runs pre-upgrade steps
- Scales down Rancher pods and deletes the ClusterRepo resources to work around Rancher upgrade issues (VZ-7053)
*/
func (r rancherComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return chartsNotUpdatedWorkaround(ctx)
}

// Install
/* Installs the Helm chart, and patches the resulting objects
- ensure Helm chart is installed
- Patch Rancher deployment with MKNOD capability
- Patch Rancher ingress with NGINX/TLS annotations
*/
func (r rancherComponent) Install(ctx spi.ComponentContext) error {
	if err := r.HelmComponent.Install(ctx); err != nil {
		return err
	}

	log := ctx.Log()
	c := ctx.Client()
	// Set MKNOD Cap on Rancher deployment
	if err := patchRancherDeployment(c); err != nil {
		return log.ErrorfThrottledNewErr("Error patching Rancher deployment: %s", err.Error())
	}
	log.Debugf("Patched Rancher deployment to support MKNOD")
	// Annotate Rancher ingress for NGINX/TLS
	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return log.ErrorfThrottledNewErr("Error patching Rancher ingress: %s", err.Error())
	}
	log.Debugf("Patched Rancher ingress")

	return nil
}

// IsReady component check
func (r rancherComponent) IsReady(ctx spi.ComponentContext) bool {
	if r.HelmComponent.IsReady(ctx) {
		return isRancherReady(ctx)
	}
	return false
}

// PostInstall
/* Additional setup for Rancher after the component is installed
- Create the Rancher admin secret if it does not already exist
- Retrieve the Rancher admin password
- Retrieve the Rancher hostname
- Set the Rancher server URL using the admin password and the hostname
*/
func (r rancherComponent) PostInstall(ctx spi.ComponentContext) error {
	c := ctx.Client()
	vz := ctx.EffectiveCR()
	log := ctx.Log()

	if err := createAdminSecretIfNotExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Error creating Rancher admin secret: %s", err.Error())
	}
	password, err := common.GetAdminSecret(c)
	if err != nil {
		return log.ErrorfThrottledNewErr("Error getting Rancher admin secret: %s", err.Error())
	}
	rancherHostName, err := getRancherHostname(c, vz)
	if err != nil {
		return log.ErrorfThrottledNewErr("Error getting Rancher hostname: %s", err.Error())
	}

	rest, err := common.NewClient(c, rancherHostName, password)
	if err != nil {
		return log.ErrorfThrottledNewErr("Error getting Rancher client: %s", err.Error())
	}
	if err := rest.SetAccessToken(); err != nil {
		return log.ErrorfThrottledNewErr("Error setting Rancher access token: %s", err.Error())
	}
	if err := rest.PutServerURL(); err != nil {
		return log.ErrorfThrottledNewErr("Error setting Rancher server URL: %s", err.Error())
	}
	if err := removeBootstrapSecretIfExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Error removing Rancher bootstrap secret: %s", err.Error())
	}
	if err := disableFirstLogin(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed disabling first login setting: %s", err.Error())
	}
	return r.HelmComponent.PostInstall(ctx)
}

// PostUpgrade disables the rancher first-login setting and adds
// cert-manager related annotation to the rancher ingress
func (r rancherComponent) PostUpgrade(ctx spi.ComponentContext) error {
	c := ctx.Client()
	log := ctx.Log()
	if err := disableFirstLogin(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed disabling first login setting: %s", err.Error())
	}

	if err := r.HelmComponent.PostUpgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post upgrade: %s", err.Error())
	}

	return patchRancherIngress(c, ctx.EffectiveCR())
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (r rancherComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Rancher != nil {
		if ctx.EffectiveCR().Spec.Components.Rancher.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Rancher.MonitorChanges
		}
		return true
	}
	return false
}
