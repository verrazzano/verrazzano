// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
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
)

// ComponentName is the name of the component
const ComponentName = common.RancherName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = common.CattleSystem

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "rancher"

const rancherIngressClassNameKey = "ingress.ingressClassName"

type rancherComponent struct {
	helm.HelmComponent
}

var certificates = []types.NamespacedName{
	{Name: "tls-rancher-ingress", Namespace: ComponentNamespace},
}

func NewComponent() spi.Component {
	return rancherComponent{
		HelmComponent: helm.HelmComponent{
			ReleaseName:               common.RancherName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), common.RancherName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "rancher-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			Certificates:              certificates,
			Dependencies:              []string{nginx.ComponentName, certmanager.ComponentName},
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

//AppendOverrides set the Rancher overrides for Helm
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	log := ctx.Log()
	rancherHostName, err := getRancherHostname(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		return kvs, log.ErrorfThrottledNewErr("Failed retrieving Rancher hostname: %s", err.Error())
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
	kvs = append(kvs, bom.KeyValue{
		Key:   rancherIngressClassNameKey,
		Value: vzconfig.GetIngressClassName(ctx.EffectiveCR()),
	})
	return appendCAOverrides(log, kvs, ctx)
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
func appendCAOverrides(log vzlog.VerrazzanoLogger, kvs []bom.KeyValue, ctx spi.ComponentContext) ([]bom.KeyValue, error) {
	cm := ctx.EffectiveCR().Spec.Components.CertManager
	if cm == nil {
		return kvs, log.ErrorfThrottledNewErr("Failed to find certManager component in effective cr")
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
		log.ErrorfThrottledNewErr("Failed creating cattle-system namespace: %s", err.Error())
		return err
	}
	if err := copyDefaultCACertificate(log, c, vz); err != nil {
		log.ErrorfThrottledNewErr("Failed copying default CA certificate: %s", err.Error())
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
	log := ctx.Log()
	if err := r.HelmComponent.Install(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving Rancher install component: %s", err.Error())
	}
	c := ctx.Client()
	// Set MKNOD Cap on Rancher deployment
	if err := patchRancherDeployment(c); err != nil {
		return log.ErrorfThrottledNewErr("Failed patching Rancher deployment: %s", err.Error())
	}
	log.Debugf("Patched Rancher deployment to support MKNOD")
	// Annotate Rancher ingress for NGINX/TLS
	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return log.ErrorfThrottledNewErr("Failed patching Rancher ingress: %s", err.Error())
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
- Activate the oci and oke drivers
*/
func (r rancherComponent) PostInstall(ctx spi.ComponentContext) error {
	c := ctx.Client()
	log := ctx.Log()

	if err := createAdminSecretIfNotExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Failed creating Rancher admin secret: %s", err.Error())
	}

	vz := ctx.EffectiveCR()
	rest, err := configureRancherRestClient(log, c, vz)
	if err != nil {
		return err
	}

	if err := rest.PutServerURL(); err != nil {
		return log.ErrorfThrottledNewErr("Failed setting Rancher server URL: %s", err.Error())
	}

	err = activateDrivers(log, rest)
	if err != nil {
		return err
	}

	if err := removeBootstrapSecretIfExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Failed removing Rancher bootstrap secret: %s", err.Error())
	}

	if err := r.HelmComponent.PostInstall(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post install: %s", err.Error())
	}
	return nil
}

// postUninstall handles the deletion of all Rancher resources after the Helm uninstall
func (r rancherComponent) PostUninstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("Rancher postUninstall dry run")
		return nil
	}
	return postUninstall(ctx)
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

// PostUpgrade configures the Rancher rest client and activates OCI and OKE drivers in Rancher
func (r rancherComponent) PostUpgrade(ctx spi.ComponentContext) error {
	c := ctx.Client()
	log := ctx.Log()
	vz := ctx.EffectiveCR()
	rest, err := configureRancherRestClient(log, c, vz)
	if err != nil {
		return err
	}

	err = activateDrivers(log, rest)
	if err != nil {
		return err
	}

	if err := r.HelmComponent.PostUpgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post upgrade: %s", err.Error())
	}

	return nil
}

// configureRancherRestClient configures the REST rancher client and sets the access token.
func configureRancherRestClient(log vzlog.VerrazzanoLogger, c client.Client, vz *vzapi.Verrazzano) (*common.RESTClient, error) {
	password, err := common.GetAdminSecret(c)
	if err != nil {
		return nil, log.ErrorfThrottledNewErr("Failed getting Rancher admin secret: %s", err.Error())
	}

	rancherHostName, err := getRancherHostname(c, vz)
	if err != nil {
		return nil, log.ErrorfThrottledNewErr("Failed getting Rancher hostname: %s", err.Error())
	}

	rest, err := common.NewClient(c, rancherHostName, password)
	if err != nil {
		return nil, log.ErrorfThrottledNewErr("Failed getting Rancher client: %s", err.Error())
	}

	if err := rest.SetAccessToken(); err != nil {
		return nil, log.ErrorfThrottledNewErr("Failed setting Rancher access token: %s", err.Error())
	}

	return rest, nil
}

// activateDrivers activates the oci nodeDriver and oraclecontainerengine kontainerDriver
func activateDrivers(log vzlog.VerrazzanoLogger, rest *common.RESTClient) error {
	if err := rest.ActivateOCIDriver(); err != nil {
		return log.ErrorfThrottledNewErr("Failed activating OCI Driver: %s", err.Error())
	}

	if err := rest.ActivateOKEDriver(); err != nil {
		return log.ErrorfThrottledNewErr("Failed activating OKE Driver: %s", err.Error())
	}

	return nil
}
