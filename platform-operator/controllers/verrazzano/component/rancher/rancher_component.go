// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
)

const ComponentName = common.RancherName

type rancherComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return rancherComponent{
		HelmComponent: helm.HelmComponent{
			Dependencies:            []string{nginx.ComponentName, certmanager.ComponentName},
			ReleaseName:             common.RancherName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), common.RancherName),
			ChartNamespace:          "cattle-system",
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "rancher-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			IngressNames: []types.NamespacedName{
				{
					Namespace: common.CattleSystem,
					Name:      constants.RancherIngress,
				},
			},
		},
	}
}

//AppendOverrides set the Rancher overrides for Helm
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

//appendRegistryOverrides appends overrides if a custom registry is being used
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
func (r rancherComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Rancher
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// PreInstall
/* Sets up the environment for Rancher
- Create the Rancher namespaces if they are not present (cattle-namespace, rancher-operator-namespace)
- Copy TLS certificates for Rancher if using the default Verrazzano CA
- Create additional LetsEncrypt TLS certificates for Rancher if using LE
*/
func (r rancherComponent) PreInstall(ctx spi.ComponentContext) error {
	vz := ctx.EffectiveCR()
	c := ctx.Client()
	log := ctx.Log()
	if err := createRancherOperatorNamespace(log, c); err != nil {
		return err
	}
	if err := createCattleSystemNamespace(log, c); err != nil {
		return err
	}
	if err := copyDefaultCACertificate(log, c, vz); err != nil {
		return err
	}
	if err := createAdditionalCertificates(log, c, vz); err != nil {
		return err
	}
	return nil
}

//Install
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
		return err
	}
	log.Debugf("Patched Rancher deployment to support MKNOD")
	// Annotate Rancher ingress for NGINX/TLS
	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return err
	}
	log.Debugf("Patched Rancher ingress")

	return nil
}

//IsReady
/* Checks that the Rancher component is in a 'Ready' state, as defined
in the body of this function
- Wait for at least one Rancher pod to be Ready in Kubernetes
*/
func (r rancherComponent) IsReady(ctx spi.ComponentContext) bool {
	if r.HelmComponent.IsReady(ctx) {
		log := ctx.Log()
		c := ctx.Client()
		rancherDeploy := []types.NamespacedName{
			{
				Name:      common.RancherName,
				Namespace: common.CattleSystem,
			},
		}
		return status.DeploymentsReady(log, c, rancherDeploy, 1)
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
		return err
	}
	password, err := common.GetAdminSecret(c)
	if err != nil {
		return err
	}
	rancherHostName, err := getRancherHostname(c, vz)
	if err != nil {
		return err
	}

	rest, err := common.NewClient(c, rancherHostName, password)
	if err != nil {
		return err
	}
	if err := rest.SetAccessToken(); err != nil {
		return err
	}

	if err := rest.PutServerURL(); err != nil {
		return err
	}

	return r.HelmComponent.PostInstall(ctx)
}
