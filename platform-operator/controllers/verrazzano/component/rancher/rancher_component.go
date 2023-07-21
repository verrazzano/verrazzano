// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	logcmn "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = common.RancherName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = common.CattleSystem

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "rancher"

const rancherIngressClassNameKey = "ingress.ingressClassName"

const (
	// Let's Encrypt environments
	letsencryptProduction = "production"
	letsEncryptStaging    = "staging"
)

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

// AppendOverrides set the Rancher overrides for Helm
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
func appendCAOverrides(log vzlog.VerrazzanoLogger, kvs []bom.KeyValue, ctx spi.ComponentContext) ([]bom.KeyValue, error) {
	cm := ctx.EffectiveCR().Spec.Components.CertManager
	if cm == nil {
		return kvs, log.ErrorfThrottledNewErr("Failed to find certManager component in effective cr")
	}

	// Configure CA Issuer KVs
	if (cm.Certificate.Acme != vzapi.Acme{}) {
		letsEncryptEnv := cm.Certificate.Acme.Environment
		if len(letsEncryptEnv) == 0 {
			letsEncryptEnv = letsencryptProduction
		}
		kvs = append(kvs,
			bom.KeyValue{
				Key:   letsEncryptIngressClassKey,
				Value: common.RancherName,
			}, bom.KeyValue{
				Key:   letsEncryptEmailKey,
				Value: cm.Certificate.Acme.EmailAddress,
			}, bom.KeyValue{
				Key:   letsEncryptEnvironmentKey,
				Value: letsEncryptEnv,
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
		kvs = append(kvs, bom.KeyValue{
			Key:   privateCAKey,
			Value: privateCAValue,
		})
	}

	return kvs, nil
}

// IsEnabled Rancher is always enabled on admin clusters,
// and is not enabled by default on managed clusters
func (r rancherComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsRancherEnabled(effectiveCR)
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
	// Block all changes for now, particularly around storage changes
	if r.IsEnabled(old) && !r.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return r.HelmComponent.ValidateUpdateV1Beta1(old, new)
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

	return r.checkRestartRequired(ctx)
}

func (r rancherComponent) Upgrade(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := r.HelmComponent.Upgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving Rancher install component: %s", err.Error())
	}
	return r.checkRestartRequired(ctx)
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

	if err := configureAuthProviders(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher auth providers: %s", err.Error())
	}

	if err := configureUISettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher UI settings: %s", err.Error())
	}

	if err := r.HelmComponent.PostInstall(ctx); err != nil {
		return err
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

	if err := configureAuthProviders(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher auth providers: %s", err.Error())
	}

	if err := configureUISettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher UI settings: %s", err.Error())
	}

	if err := r.HelmComponent.PostUpgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post upgrade: %s", err.Error())
	}

	return patchRancherIngress(c, ctx.EffectiveCR())
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
func configureAuthProviders(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) && isKeycloakAuthEnabled(ctx.EffectiveCR()) {
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

// isKeycloakAuthEnabled checks if Keycloak as an Auth provider is enabled for Rancher
// +returns false if Keycloak component is itself disabled.
// +returns the value of the keycloakAuthEnabled attribute if it is set in rancher component of VZ CR.
// +returns true otherwise.
func isKeycloakAuthEnabled(vz *vzapi.Verrazzano) bool {
	if !vzconfig.IsKeycloakEnabled(vz) {
		return false
	}

	if vz.Spec.Components.Rancher != nil && vz.Spec.Components.Rancher.KeycloakAuthEnabled != nil {
		return *vz.Spec.Components.Rancher.KeycloakAuthEnabled
	}

	return true
}

// configureUISettings configures Rancher setting ui-pl, ui-logo-light, ui-logo-dark and ui-brand.
func configureUISettings(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := createOrUpdateUIPlSetting(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui-pl setting: %s", err.Error())
	}

	if err := createOrUpdateUILogoSetting(ctx, SettingUILogoLight, SettingUILogoLightLogoFilePath); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring %s setting for logo path %s: %s", SettingUILogoLight, SettingUILogoLightLogoFilePath, err.Error())
	}

	if err := createOrUpdateUILogoSetting(ctx, SettingUILogoDark, SettingUILogoDarkLogoFilePath); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring %s setting for logo path %s: %s", SettingUILogoDark, SettingUILogoDarkLogoFilePath, err.Error())
	}

	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIBrand}, common.GVKSetting, map[string]interface{}{"value": SettingUIBrandValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui-brand setting: %s", err.Error())
	}

	return nil
}

// checkRestartRequired Restarts the Rancher deployment if necessary; at present this is required when the
// private CA bundle/secret is configured and updated, but the Rancher deployment hasn't been rolled to pick it up
func (r rancherComponent) checkRestartRequired(ctx spi.ComponentContext) error {
	if r.isRancherDeploymentUpdateInProgress(ctx) {
		// The rancher pods are already in the process of being updated
		ctx.Log().Debugf("Rancher deployment update already in progress, skipping restart check")
		return nil
	}
	privateCABundleInSync, err := r.isPrivateCABundleInSync(ctx)
	if err != nil {
		return err
	}
	if privateCABundleInSync {
		// Rancher pods have the latest tls-ca bundle reflected in the Settings object, nothing to do
		return nil
	}
	// The Rancher pods' "cacerts" Settings value is out of sync with tls-ca, do a rolling restart of the Rancher pods
	// to pick up the new bundle
	ctx.Log().Progressf("Rancher private CA bundle drift detected, performing a rolling restart of the Rancher deployment")
	return restartRancherDeployment(ctx.Log(), ctx.Client())
}

func (r rancherComponent) isRancherDeploymentUpdateInProgress(ctx spi.ComponentContext) bool {
	rancherDeployment := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	return !ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), rancherDeployment, 1, "rancher")
}

// isPrivateCABundleInSync If the tls-ca private CA bundle secret is present, verify that the bundle in the secret is
// in sync with the "cacerts" Settings object being used by pods; if they are not in sync it is an indicator that we
// need to roll the Rancher deployment
func (r rancherComponent) isPrivateCABundleInSync(ctx spi.ComponentContext) (bool, error) {
	currentCABundleInSecret, found, err := r.getCurrentCABundleSecretsValue(ctx, rancherTLSSecretName, caCertsPem)
	if err != nil {
		return false, err
	}
	if !found {
		return true, nil
	}
	cacertsSettingsValue, err := getSettingValue(ctx.Client(), SettingCACerts)
	if err != nil {
		return false, err
	}
	return cacertsSettingsValue == currentCABundleInSecret, nil
}

// getCurrentCABundleSecretsValue Returns the current CA bundle stored in the Rancher tls-ca secret as a trimmed string
func (r rancherComponent) getCurrentCABundleSecretsValue(ctx spi.ComponentContext, secretName string, key string) (string, bool, error) {
	tlsCASecret, err := getSecret(ComponentNamespace, secretName)
	if err != nil {
		if clipkg.IgnoreNotFound(err) != nil {
			return "", false, err
		}
		ctx.Log().Debugf("%s secret not defined, skipping restart", secretName)
		return "", false, nil
	}
	currentCACerts, found := tlsCASecret.Data[key]
	if !found {
		return "", false, ctx.Log().ErrorfThrottledNewErr("Did not find %s key in % secret", key,
			clipkg.ObjectKeyFromObject(tlsCASecret))
	}
	return strings.TrimSpace(string(currentCACerts)), true, nil
}

func restartRancherDeployment(log vzlog.VerrazzanoLogger, c clipkg.Client) error {
	deployment := appsv1.Deployment{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.RancherSystemNamespace,
		Name: ComponentName}, &deployment); err != nil {
		if kerrs.IsNotFound(err) {
			log.Debugf("Rancher deployment %s/%s not found, nothing to do",
				vzconst.RancherSystemNamespace, ComponentName)
			return nil
		}
		log.ErrorfThrottled("Failed getting Rancher deployment %s/%s to restart pod: %v",
			vzconst.RancherSystemNamespace, ComponentName, err)
		return err
	}

	// annotate the deployment to do a restart of the pod
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().String()

	if err := c.Update(context.TODO(), &deployment); err != nil {
		return logcmn.ConflictWithLog(fmt.Sprintf("Failed updating deployment %s/%s", deployment.Namespace, deployment.Name),
			err, log.GetRootZapLogger())
	}
	log.Debugf("Updated Rancher deployment %s/%s with restart annotation to force a pod restart",
		deployment.Namespace, deployment.Name)
	return nil
}

func getSecret(namespace string, name string) (*v1.Secret, error) {
	v1Client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return nil, err
	}
	return v1Client.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}
