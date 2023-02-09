// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"github.com/gertd/go-pluralize"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/monitor"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

// ComponentName is the name of the component
const ComponentName = common.RancherName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = common.CattleSystem

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "rancher"

const rancherIngressClassNameKey = "ingress.ingressClassName"

// rancherImageSubcomponent is the name of the subcomponent for the additional Rancher images
const rancherImageSubcomponent = "additional-rancher"

// cattleShellImageName is the name of the shell image used for the shell override special case
const cattleShellImageName = "rancher-shell"

// cattleUIEnvName is the environment variable name to set for the Rancher dashboard
const cattleUIEnvName = "CATTLE_UI_OFFLINE_PREFERRED"

// Environment variables for the Rancher images
// format: imageName: baseEnvVar
var imageEnvVars = map[string]string{
	"rancher-fleet":       "FLEET_IMAGE",
	"rancher-fleet-agent": "FLEET_AGENT_IMAGE",
	"rancher-shell":       "CATTLE_SHELL_IMAGE",
	"rancher-webhook":     "RANCHER_WEBHOOK_IMAGE",
	"rancher-gitjob":      "GITJOB_IMAGE",
}

type envVar struct {
	Name      string
	Value     string
	SetString bool
}

type rancherComponent struct {
	helm.HelmComponent

	// internal monitor object for running the Rancher uninstall tool in the background
	monitor monitor.BackgroundProcessMonitor
}

var certificates = []types.NamespacedName{
	{Name: "tls-rancher-ingress", Namespace: ComponentNamespace},
}

// For use to override during unit tests
type checkClusterProvisionedFuncSig func(client corev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error)

var checkClusterProvisionedFunc checkClusterProvisionedFuncSig = checkClusterProvisioned

func SetCheckClusterProvisionedFunc(newFunc checkClusterProvisionedFuncSig) {
	checkClusterProvisionedFunc = newFunc
}
func SetDefaultCheckClusterProvisionedFunc() {
	checkClusterProvisionedFunc = checkClusterProvisioned
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
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName, certmanager.ComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
					{
						Name:      rancherWebhookDeployment,
						Namespace: ComponentNamespace,
					},
					{
						Name:      fleetAgentDeployment,
						Namespace: FleetLocalSystemNamespace,
					},
					{
						Name:      fleetControllerDeployment,
						Namespace: FleetSystemNamespace,
					},
					{
						Name:      gitjobDeployment,
						Namespace: FleetSystemNamespace,
					},
				},
			},
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.RancherIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
		monitor: &monitor.BackgroundProcessMonitorType{ComponentName: ComponentName},
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
	kvs, err = appendImageOverrides(ctx, kvs)
	if err != nil {
		return kvs, err
	}
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

// appendImageOverrides creates overrides to set the pod environment variables for the image overrides
func appendImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get the bom file for the Rancher image overrides: %v", err)
	}

	registryOverride := os.Getenv(constants.RegistryOverrideEnvVar)
	subcomponent, err := bomFile.GetSubcomponent(rancherImageSubcomponent)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get the subcomponent %s from the bom: %v", rancherImageSubcomponent, err)
	}
	images := subcomponent.Images

	var envList []envVar
	for _, image := range images {
		imEnvVar, ok := imageEnvVars[image.ImageName]
		// skip the images that are not included in the override map
		if !ok {
			continue
		}

		// if there is a registry override set, it will be communicated to Rancher using the "systemDefaultRegistry" helm value,
		// otherwise we prepend the image here with the registry value from the BOM
		var registry = ""
		if registryOverride == "" {
			registry = bomFile.ResolveRegistry(subcomponent, image) + "/"
		}

		fullImageName := fmt.Sprintf("%s%s/%s", registry, subcomponent.Repository, image.ImageName)
		// For the shell image, we need to combine to one env var
		if image.ImageName == cattleShellImageName {
			envList = append(envList, envVar{Name: imEnvVar, Value: fmt.Sprintf("%s:%s", fullImageName, image.ImageTag), SetString: false})
			continue
		}
		tagEnvVar := imEnvVar + "_TAG"
		envList = append(envList, envVar{Name: imEnvVar, Value: fullImageName, SetString: false})
		envList = append(envList, envVar{Name: tagEnvVar, Value: image.ImageTag, SetString: false})
	}

	// For the Rancher UI, we need to update this final env var
	envList = append(envList, envVar{Name: cattleUIEnvName, Value: "true", SetString: true})

	return createEnvVars(kvs, envList), nil
}

// createEnvVars takes in a list of env arguments and creates the extraEnv override arguments
func createEnvVars(kvs []bom.KeyValue, envList []envVar) []bom.KeyValue {
	envPos := 0
	for _, env := range envList {
		kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("extraEnv[%d].name", envPos), Value: env.Name})
		kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("extraEnv[%d].value", envPos), Value: env.Value, SetString: env.SetString})
		envPos++
	}
	return kvs
}

// IsEnabled Rancher is always enabled on admin clusters,
// and is not enabled by default on managed clusters
func (r rancherComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsRancherEnabled(effectiveCR)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
// and also if the rancher is already installed by some other source by checking the namespace labels.
func (r rancherComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	if err := checkExistingRancher(vz); err != nil {
		return err
	}
	return r.HelmComponent.ValidateInstall(vz)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
// and also if the rancher is already installed by some other source by checking the namespace labels.
func (r rancherComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	if err := checkExistingRancher(vz); err != nil {
		return err
	}
	return r.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (r rancherComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of component
	if r.IsEnabled(old) && !r.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return r.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (r rancherComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow disabling of component
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
	return r.HelmComponent.PreInstall(ctx)
}

// PreUpgrade
/* Runs pre-upgrade steps
- Scales down Rancher pods and deletes the ClusterRepo resources to work around Rancher upgrade issues (VZ-7053)
*/
func (r rancherComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := chartsNotUpdatedWorkaround(ctx); err != nil {
		return err
	}
	return r.HelmComponent.PreUpgrade(ctx)
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

	return nil
}

// IsReady component check
func (r rancherComponent) IsReady(ctx spi.ComponentContext) bool {
	if r.HelmComponent.IsReady(ctx) {
		return r.isRancherReady(ctx)
	}
	return false
}

// PostInstall
/* Additional setup for Rancher after the component is installed
- Label Rancher Component Namespaces
- Create the Rancher admin secret if it does not already exist
- Retrieve the Rancher admin password
- Retrieve the Rancher hostname
- Set the Rancher server URL using the admin password and the hostname
- Activate the OCI and OKE drivers
*/
func (r rancherComponent) PostInstall(ctx spi.ComponentContext) error {
	c := ctx.Client()
	log := ctx.Log()

	if err := labelNamespace(c); err != nil {
		return log.ErrorfThrottledNewErr("failed labelling namespace the for Rancher component: %s", err.Error())
	}
	log.Debugf("Rancher component namespaces labelled")

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

	if err := configureUISettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher UI settings: %s", err.Error())
	}

	if err := r.HelmComponent.PostInstall(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post install: %s", err.Error())
	}
	return nil
}

// PostUninstall handles the deletion of all Rancher resources after the Helm uninstall
func (r rancherComponent) PostUninstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("Rancher postUninstall dry run")
		return nil
	}
	return postUninstall(ctx, r.monitor)
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

// ConfigureAuthProviders
// +configures Keycloak as OIDC provider for Rancher.
// +creates or updates default user verrazzano.
// +creates or updated the verrazzano cluster user
// +creates or updates admin clusterRole binding for  user verrazzano.
// +disables first login setting to disable prompting for password on first login.
// +enables or disables Keycloak Auth provider.
func ConfigureAuthProviders(ctx spi.ComponentContext) error {
	if vzcr.IsKeycloakEnabled(ctx.EffectiveCR()) &&
		isKeycloakAuthEnabled(ctx.EffectiveCR()) &&
		vzcr.IsRancherEnabled(ctx.EffectiveCR()) {

		ctx.Log().Oncef("Configuring Keycloak as a Rancher authentication provider")
		if err := configureKeycloakOIDC(ctx); err != nil {
			return err
		}

		if err := createOrUpdateRancherUser(ctx); err != nil {
			return err
		}

		if err := createOrUpdateRoleTemplates(ctx); err != nil {
			return err
		}

		if err := createOrUpdateClusterRoleTemplateBindings(ctx); err != nil {
			return err
		}

		if err := disableFirstLogin(ctx); err != nil {
			return err
		}
	}
	return nil
}

// createOrUpdateRoleTemplates creates or updates the verrazzano-admin and verrazzano-monitor RoleTemplates
func createOrUpdateRoleTemplates(ctx spi.ComponentContext) error {
	if err := CreateOrUpdateRoleTemplate(ctx, VerrazzanoAdminRoleName); err != nil {
		return err
	}

	return CreateOrUpdateRoleTemplate(ctx, VerrazzanoMonitorRoleName)
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
	if !vzcr.IsKeycloakEnabled(vz) {
		return false
	}

	if vz.Spec.Components.Rancher != nil && vz.Spec.Components.Rancher.KeycloakAuthEnabled != nil {
		return *vz.Spec.Components.Rancher.KeycloakAuthEnabled
	}

	return true
}

// configureUISettings configures Rancher setting ui-pl, ui-logo-light, ui-logo-dark, ui-primary-color, ui-link-color and ui-brand.
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

	if err := createOrUpdateUIColorSettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui color settings: %s", err.Error())
	}

	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIBrand}, common.GVKSetting, map[string]interface{}{"value": SettingUIBrandValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui-brand setting: %s", err.Error())
	}

	return nil
}

// checkExistingRancher checks if there is already an existing Rancher or not
func checkExistingRancher(vz runtime.Object) error {
	if !vzcr.IsRancherEnabled(vz) {
		return nil
	}

	provisioned, err := IsClusterProvisionedByRancher()
	if err != nil {
		return err
	}
	// If the k8s cluster was provisioned by Rancher then don't check for Rancher namespaces.
	// A Rancher provisioned cluster will have Rancher namespaces.
	if provisioned {
		return nil
	}
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}

	ns, err := client.Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil && !kerrs.IsNotFound(err) {
		return err
	}
	if err = common.CheckExistingNamespace(ns.Items, isRancherNamespace); err != nil {
		return err
	}
	return nil
}

// IsClusterProvisionedByRancher checks if the Kubernetes cluster was provisioned by Rancher.
func IsClusterProvisionedByRancher() (bool, error) {
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return false, err
	}
	dynClient, err := k8sutil.GetDynamicClientFunc()
	if err != nil {
		return false, err
	}

	return checkClusterProvisionedFunc(client, dynClient)
}

// checkClusterProvisioned checks if the Kubernetes cluster was provisioned by Rancher.
func checkClusterProvisioned(client corev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error) {
	// Check for the "local" namespace.
	ns, err := client.Namespaces().Get(context.TODO(), ClusterLocal, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Find the management.cattle.io Cluster resource and check if the provider.cattle.io label exists.
	for _, ownerRef := range ns.OwnerReferences {
		group, version := controllers.ConvertAPIVersionToGroupAndVersion(ownerRef.APIVersion)
		if group == common.APIGroupRancherManagement && ownerRef.Kind == ClusterKind {
			resource := schema.GroupVersionResource{
				Group:    group,
				Version:  version,
				Resource: pluralize.NewClient().Plural(strings.ToLower(ownerRef.Kind)),
			}
			u, err := dynClient.Resource(resource).Namespace("").Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			labels := u.GetLabels()
			_, ok := labels[ProviderCattleIoLabel]
			if ok {
				return true, nil
			}
			return false, nil
		}
	}

	return false, nil
}

// createOrUpdateRancherUser create or update the new Rancher user mapped to Keycloak user verrazzano
func createOrUpdateRancherUser(ctx spi.ComponentContext) error {
	vzUser, err := keycloak.GetVerrazzanoUserFromKeycloak(ctx)
	if err != nil {
		return ctx.Log().ErrorfThrottledNewErr("failed configuring Rancher user, unable to fetch verrazzano user id from Keycloak: %s", err.Error())
	}
	rancherUsername, err := getRancherUsername(ctx, vzUser)
	if err != nil {
		return err
	}
	if err = createOrUpdateRancherVerrazzanoUser(ctx, vzUser, rancherUsername); err != nil {
		return err
	}

	if err = createOrUpdateRancherVerrazzanoUserGlobalRoleBinding(ctx, rancherUsername); err != nil {
		return err
	}
	return nil
}
