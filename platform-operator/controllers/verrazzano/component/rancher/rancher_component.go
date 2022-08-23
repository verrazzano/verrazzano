// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"os"
	"path/filepath"
	"strconv"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
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

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (r rancherComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
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
- Activate the OCI and OKE drivers
*/
func (r rancherComponent) PostInstall(ctx spi.ComponentContext) error {
	c := ctx.Client()
	log := ctx.Log()

	if err := createAdminSecretIfNotExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Failed creating Rancher admin secret: %s", err.Error())
	}

	vz := ctx.EffectiveCR()
	rancherHostName, err := getRancherHostname(c, vz)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting Rancher hostname: %s", err.Error())
	}

	if err := putServerURL(log, c, fmt.Sprintf("https://%s", rancherHostName)); err != nil {
		return log.ErrorfThrottledNewErr("Failed setting Rancher server URL: %s", err.Error())
	}

	err = activateDrivers(log, c)
	if err != nil {
		return err
	}

	if err := removeBootstrapSecretIfExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Failed removing Rancher bootstrap secret: %s", err.Error())
	}

	if err := configureAuthProviders(ctx, false); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher auth providers: %s", err.Error())
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
	err := activateDrivers(log, c)
	if err != nil {
		return err
	}

	if err := configureAuthProviders(ctx, true); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher auth providers: %s", err.Error())
	}

	if err := r.HelmComponent.PostUpgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post upgrade: %s", err.Error())
	}

	return nil
}

// activateDrivers activates the nodeDriver oci and oraclecontainerengine kontainerDriver
func activateDrivers(log vzlog.VerrazzanoLogger, c client.Client) error {
	err := activateOCIDriver(log, c)
	if err != nil {
		return err
	}

	err = activatOKEDriver(log, c)
	if err != nil {
		return err
	}

	return nil
}

// configureAuthProviders
// +configures Keycloak as OIDC provider for Rancher.
// +creates or updates default user verrazzano.
// +creates or updates admin clusterRole binding for  user verrazzano.
// +disables first login setting to disable prompting for password on first login.
// +enables or disables Keycloak Auth provider.
func configureAuthProviders(ctx spi.ComponentContext, isUpgrade bool) error {
	log := ctx.Log()
	if err := configureKeycloakOIDC(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring keycloak oidc provider: %s", err.Error())
	}

	if err := createOrUpdateRancherVerrazzanoUser(ctx); err != nil {
		return err
	}

	if err := createOrUpdateRancherVerrazzanoUserGlobalRoleBinding(ctx); err != nil {
		return err
	}

	if err := createOrUpdateRoleTemplates(ctx); err != nil {
		return err
	}

	if err := createOrUpdateClusterRoleTemplateBindings(ctx); err != nil {
		return err
	}

	if err := disableFirstLogin(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed disabling first login setting: %s", err.Error())
	}

	if err := toggleKeycloakAuthProvider(ctx, isUpgrade); err != nil {
		return log.ErrorfThrottledNewErr("failed enabling or disbling auth providers: %s", err.Error())
	}

	return nil
}

// createOrUpdateRoleTemplates creates or updates the verrazzano-admin and verrazzano-monitor RoleTemplates
func createOrUpdateRoleTemplates(ctx spi.ComponentContext) error {
	if err := createOrUpdateRoleTemplate(ctx, VerrazzanoAdminRoleName); err != nil {
		return err
	}

	return createOrUpdateRoleTemplate(ctx, VerrazzanoMonitorRoleName)
}

// createOrUpdateClusterRoleTemplateBindings creates or updates the CRTBs for the verrazzano-admins and verrazzano-monitors groups
func createOrUpdateClusterRoleTemplateBindings(ctx spi.ComponentContext) error {
	for _, grp := range GroupRolePairs {
		if err := createOrUpdateClusterRoleTemplateBinding(ctx, grp[ClusterRoleKey], grp[GroupKey]); err != nil {
			return err
		}
	}

	return nil
}

func parseVersions(ctx spi.ComponentContext, isUpgrade bool) (*semver.SemVersion, *semver.SemVersion, *semver.SemVersion, error) {
	ver140, err := semver.NewSemVersion("v" + constants.VerrazzanoVersion1_4_0)
	if err != nil {
		return nil, nil, nil, err
	}

	var vzSourceVersion, vzTargetVersion *semver.SemVersion
	vz := ctx.ActualCR()

	if vz.Spec.Version != "" {
		vzTargetVersion, err = semver.NewSemVersion(vz.Spec.Version)
		if err != nil {
			return nil, nil, nil, ctx.Log().ErrorfNewErr("Failed Rancher post-%v: Invalid Verrazzano spec version: %v", formatBool(isUpgrade, "upgrade", "install"), err)
		}
	}

	if vz.Status.Version != "" {
		vzSourceVersion, err = semver.NewSemVersion(vz.Status.Version)
		if err != nil {
			return nil, nil, nil, ctx.Log().ErrorfNewErr("Failed Rancher post-%v: Invalid Verrazzano status version: %v", formatBool(isUpgrade, "upgrade", "install"), err)
		}
	}

	return ver140, vzSourceVersion, vzTargetVersion, nil
}

func authProviderForRancherImplemented(ver140 *semver.SemVersion, vzSourceVersion *semver.SemVersion, vzTargetVersion *semver.SemVersion, isUpgrade bool) (bool, error) {
	var version *semver.SemVersion
	if isUpgrade {
		version = vzTargetVersion
	} else {
		version = vzSourceVersion
	}

	return version == nil || version.IsGreaterThanOrEqualTo(ver140), nil
}

// isKeycloakAuthEnabled checks if Keycloak as an Auth provider is enabled for Rancher
// +returns false if Keycloak component is itself disabled.
// +returns true when the keycloakAuthEnabled attribute is set to true in rancher component of VZ CR.
// +when keycloakAuthEnabled is not specified, returns true for new installs and in case of upgrades between versions>=1.4.
// +returns false otherwise.
func isKeycloakAuthEnabled(isUpgrade bool, vz *vzapi.Verrazzano, ver140 *semver.SemVersion, vzSourceVersion *semver.SemVersion, vzTargetVersion *semver.SemVersion) bool {
	if !vzconfig.IsKeycloakEnabled(vz) {
		return false
	}

	if vz.Spec.Components.Rancher != nil && vz.Spec.Components.Rancher.KeycloakAuthEnabled != nil && *vz.Spec.Components.Rancher.KeycloakAuthEnabled {
		return true
	}

	if vz.Spec.Components.Rancher == nil || vz.Spec.Components.Rancher.KeycloakAuthEnabled == nil {
		if !isUpgrade {
			return true
		}

		if vzSourceVersion != nil && vzSourceVersion.IsGreaterThanOrEqualTo(ver140) && vzTargetVersion != nil && vzTargetVersion.IsGreaterThanOrEqualTo(vzSourceVersion) {
			return true
		}

	}

	return false
}

// toggleKeycloakAuthProvider enables/disables Keycloak as Auth provider
func toggleKeycloakAuthProvider(ctx spi.ComponentContext, isUpgrade bool) error {
	ver140, vzSourceVersion, vzTargetVersion, err := parseVersions(ctx, isUpgrade)
	if err != nil {
		return err
	}

	checkAuthProvider, err := authProviderForRancherImplemented(ver140, vzSourceVersion, vzTargetVersion, isUpgrade)
	if err != nil {
		return err
	}

	if !checkAuthProvider {
		ctx.Log().Debug("Rancher Keycloak AuthProvider not implemented")
		return nil
	}

	log := ctx.Log()
	vz := ctx.ActualCR()
	enableKeycloak := isKeycloakAuthEnabled(isUpgrade, vz, ver140, vzSourceVersion, vzTargetVersion)
	if err := disableOrEnableAuthProvider(ctx, AuthConfigKeycloak, enableKeycloak); err != nil {
		return log.ErrorfThrottledNewErr("failed to %s keycloak oidc auth provider, error: %s", formatBool(enableKeycloak, "enable", "disable"), err.Error())
	}

	return nil

}

func formatBool(isTrue bool, trueValue string, falseValue string) string {
	if isTrue {
		return trueValue
	}
	return falseValue
}
