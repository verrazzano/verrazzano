// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"path/filepath"
	"strconv"
)

type rancherComponent struct {
	helm.HelmComponent
	ingressIp string
}

func NewComponent() spi.Component {
	return rancherComponent{
		HelmComponent: helm.HelmComponent{
			Dependencies:            []string{nginx.ComponentName},
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          "cattle-system",
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "rancher-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
		},
	}
}

//AppendOverrides set the Rancher overrides for Helm
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs = appendRegistryOverrides(kvs)
	return appendCAOverrides(kvs, ctx)
}

//appendRegistryOverrides appends overrides if a custom registry is being used
func appendRegistryOverrides(kvs []bom.KeyValue) []bom.KeyValue {
	// If using external registry, add registry overrides to rancher
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
		}, bom.KeyValue{
			Key:   useBundledSystemChartKey,
			Value: "true",
		})
	}

	return kvs
}

//appendCAOverrides sets overrides for CA Issuers, ACME or CA.
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
				Value: ComponentName,
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
				Value: "true",
			})
		}
	}

	return kvs, nil
}

// IsEnabled Rancher is always enabled on admin clusters,
// and never enabled on managed clusters
func (r rancherComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Rancher
	if comp != nil && comp.Enabled != nil {
		return *comp.Enabled
	}
	return r.HelmComponent.IsEnabledFunc(ctx)
}

// PreInstall
/* Sets up the environment for Rancher
1. Create the Rancher namespace if it does not exist
2. Create Rancher TLS Certificate(s) as appropriate for the environment
*/
func (r rancherComponent) PreInstall(ctx spi.ComponentContext) error {
	vz := ctx.EffectiveCR()
	c := ctx.Client()
	log := ctx.Log()
	if err := createCattleSystemNamespaceIfNotExists(log, c); err != nil {
		return err
	}
	if err := copyDefaultCACertificate(log, c, vz); err != nil {
		return err
	}
	if err := createAdditionalCertificates(log, vz); err != nil {
		return err
	}
	return nil
}

//Install installs the Helm chart, and patches the resulting objects
func (r rancherComponent) Install(ctx spi.ComponentContext) error {
	if err := r.HelmComponent.Install(ctx); err != nil {
		return err
	}

	log := ctx.Log()
	c := ctx.Client()
	// Set MKNOD Cap on Rancher deployment
	if err := patchRancherDeployment(c); err != nil {
		return err
	}
	log.Infof("Pached Rancher deployment to support MKNOD")
	// Annotate Rancher ingress for NGINX/TLS
	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return err
	}
	log.Infof("Patched Rancher ingress")

	return nil
}

func (r rancherComponent) IsReady(ctx spi.ComponentContext) bool {
	if r.HelmComponent.IsReady(ctx) {
		log := ctx.Log()
		c := ctx.Client()
		// Try to retrieve the Rancher ingress IP
		_, err := getRancherIngressIP(log, c)
		if err != nil {
			log.Errorf("Rancher IsReady: Failed to get Ingress IP: %s", err)
			return false
		}
		return true
	}

	return false
}

// PostInstall
// Configures the environment after Rancher has been created
func (r rancherComponent) PostInstall(ctx spi.ComponentContext) error {
	c := ctx.Client()
	vz := ctx.EffectiveCR()
	log := ctx.Log()

	if err := createAdminSecretIfNotExists(log, c); err != nil {
		return err
	}
	ip, err := getRancherIngressIP(log, c)
	if err != nil {
		return err
	}
	dnsSuffix, err := nginx.GetDNSSuffix(c, vz)
	if err != nil {
		return err
	}
	rancherHostname := fmt.Sprintf("%s.%s.%s", ComponentName, vz.Spec.EnvironmentName, dnsSuffix)
	password, err := getAdminPassword(c)
	if err != nil {
		return err
	}
	if err := setServerURL(log, password, rancherHostname); err != nil {
		return err
	}
	if err := patchAgents(log, c, rancherHostname, ip); err != nil {
		return err
	}
	return nil
}
