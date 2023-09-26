// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gertd/go-pluralize"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/certs"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	logcmn "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/monitor"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sversionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = common.RancherName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = common.CattleSystem

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "rancher"

// CattleGlobalDataNamespace is the multi-cluster namespace for verrazzano
const CattleGlobalDataNamespace = "cattle-global-data"

const rancherIngressClassNameKey = "ingress.ingressClassName"

// rancherImageSubcomponent is the name of the subcomponent for the additional Rancher images
const rancherImageSubcomponent = "additional-rancher"

// cattleShellImageName is the name of the shell image used for the shell override special case
const cattleShellImageName = "rancher-shell"

// cattleUIEnvName is the environment variable name to set for the Rancher dashboard
const cattleUIEnvName = "CATTLE_UI_OFFLINE_PREFERRED"

// fluentbitFilterAndParserTemplate is the template name that consists Fluentbit Filter and Parser resource for Istio.
const fluentbitFilterAndParserTemplate = "rancher-filter-parser.yaml"

// clusterProvisioner is the configmap indicating the kontainer driver that provisioned the cluster
const clusterProvisioner = "cluster-provisioner"

// Environment variables for the Rancher images
// format: imageName: baseEnvVar
var imageEnvVars = map[string]string{}
var imageEnvVarsMutex = &sync.Mutex{}

var getKubernetesClusterVersion = getKubernetesVersion

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
type checkProvisionedFuncSig func(client corev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error)

var checkClusterProvisionedFunc checkProvisionedFuncSig = checkClusterProvisioned

func SetCheckClusterProvisionedFunc(newFunc checkProvisionedFuncSig) {
	checkClusterProvisionedFunc = newFunc
}
func SetDefaultCheckClusterProvisionedFunc() {
	checkClusterProvisionedFunc = checkClusterProvisioned
}

var checkContainerDriverProvisionedFunc checkProvisionedFuncSig = checkContainerDriverProvisioned

func SetCheckContainerDriverProvisionedFunc(newFunc checkProvisionedFuncSig) {
	checkContainerDriverProvisionedFunc = newFunc
}
func SetDefaultCheckContainerDriverProvisionedFunc() {
	checkContainerDriverProvisionedFunc = checkContainerDriverProvisioned
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
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, cmconstants.ClusterIssuerComponentName, fluentoperator.ComponentName},
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

// initializeImageEnvVars - initialize the translation table for image names to environment variables
func initializeImageEnvVars(imageEnvMap map[string]string) error {
	// Synchronize so that map is only written once
	imageEnvVarsMutex.Lock()
	defer imageEnvVarsMutex.Unlock()
	if len(imageEnvMap) > 0 {
		return nil
	}

	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return fmt.Errorf("Failed to get the bom file for the Rancher image overrides: %v", err)
	}

	subcomponent, err := bomFile.GetSubcomponent(rancherImageSubcomponent)
	if err != nil {
		return fmt.Errorf("Failed to get the subcomponent %s from the bom: %v", rancherImageSubcomponent, err)
	}

	for _, image := range subcomponent.Images {
		if strings.Contains(image.ImageName, "rancher-fleet-agent") {
			imageEnvMap[image.ImageName] = "FLEET_AGENT_IMAGE"
		} else if strings.Contains(image.ImageName, "rancher-fleet") {
			imageEnvMap[image.ImageName] = "FLEET_IMAGE"
		} else if strings.Contains(image.ImageName, "rancher-shell") {
			imageEnvMap[image.ImageName] = "CATTLE_SHELL_IMAGE"
		} else if strings.Contains(image.ImageName, "rancher-webhook") {
			imageEnvMap[image.ImageName] = "RANCHER_WEBHOOK_IMAGE"
		} else if strings.Contains(image.ImageName, "rancher-gitjob") {
			imageEnvMap[image.ImageName] = "GITJOB_IMAGE"
		}
	}
	return nil
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
	kvs, err = appendPSPEnabledOverrides(ctx, kvs)
	if err != nil {
		return kvs, err
	}
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

// appendCAOverrides sets overrides for CA Issuers, LetsEncrypt or CA.
func appendCAOverrides(log vzlog.VerrazzanoLogger, kvs []bom.KeyValue, ctx spi.ComponentContext) ([]bom.KeyValue, error) {

	cm := ctx.EffectiveCR().Spec.Components.ClusterIssuer
	if cm == nil {
		return kvs, log.ErrorfThrottledNewErr("Failed to find clusterIssuer component in effective cr")
	}

	isLetsEncryptIssuer, err := cm.IsLetsEncryptIssuer()
	if err != nil {
		return kvs, err
	}

	// Always disable this as we're no longer using this helm value for Let's Encrypt staging anymore
	kvs = append(kvs, bom.KeyValue{
		Key: additionalTrustedCAsKey,
		// by default disable this explicitly so upgrade works, as all untrusted CAs for Rancher SSO are
		// managed via tls-ca; can still be overridden by users via custom Helm overrides
		Value: "false",
	})

	// Configure CA Issuer KVs
	if isLetsEncryptIssuer {
		letsEncryptEnv := cm.LetsEncrypt.Environment
		if len(letsEncryptEnv) == 0 {
			letsEncryptEnv = vzconst.LetsEncryptProduction
		}
		kvs = append(kvs,
			bom.KeyValue{
				Key:   letsEncryptIngressClassKey,
				Value: common.RancherName,
			}, bom.KeyValue{
				Key:   letsEncryptEmailKey,
				Value: cm.LetsEncrypt.EmailAddress,
			}, bom.KeyValue{
				Key:   letsEncryptEnvironmentKey,
				Value: letsEncryptEnv,
			}, bom.KeyValue{
				Key:   ingressTLSSourceKey,
				Value: letsEncryptTLSSource,
			})
	}

	// Configure private issuer bundle if necessary
	if isPrivateIssuer, _ := certs.IsPrivateIssuer(cm); isPrivateIssuer {
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
	if err := initializeImageEnvVars(imageEnvVars); err != nil {
		return kvs, err
	}
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

// appendPSPEnabledOverrides appends overrides to disable PSP if the K8S version is 1.25 or above
func appendPSPEnabledOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	version, err := getKubernetesClusterVersion()
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get the kubernetes version: %v", err)
	}
	k8sVersion, err := k8sversionutil.ParseSemantic(version)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to parse Kubernetes version %q: %v", version, err)
	}
	// If K8s version is 1.25 or above, set pspEnabled to false
	pspDisabledVersion := k8sversionutil.MustParseSemantic("1.25.0-0")
	if k8sVersion.AtLeast(pspDisabledVersion) {
		kvs = append(kvs, bom.KeyValue{
			Key:   pspEnabledKey,
			Value: "false",
		})
	}
	return kvs, nil
}

// getKubernetesVersion returns the version of Kubernetes cluster in which operator is deployed
func getKubernetesVersion() (string, error) {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		return "", fmt.Errorf("Failed to get kubernetes client config %v", err.Error())
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("Failed to get kubernetes client %v", err.Error())
	}

	versionInfo, err := client.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("Failed to get kubernetes version %v", err.Error())
	}

	return versionInfo.String(), nil
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
	if err := copyPrivateCABundles(log, c, vz); err != nil {
		ctx.Log().ErrorfThrottledNewErr("Failed setting up private CA bundles for Rancher: %s", err.Error())
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
	if err := copyPrivateCABundles(ctx.Log(), ctx.Client(), ctx.EffectiveCR()); err != nil {
		ctx.Log().ErrorfThrottledNewErr("Failed setting up private CA bundles for Rancher: %s", err.Error())
		return err
	}
	return r.HelmComponent.PreUpgrade(ctx)
}

// Install
/* Installs the Helm chart, and patches the resulting objects
- ensure Helm chart is installed
- Patch Rancher ingress with NGINX/TLS annotations
*/
func (r rancherComponent) Install(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := r.HelmComponent.Install(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving Rancher install component: %s", err.Error())
	}
	c := ctx.Client()
	// Annotate Rancher ingress for NGINX/TLS
	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return log.ErrorfThrottledNewErr("Failed patching Rancher ingress: %s", err.Error())
	}
	log.Debugf("Patched Rancher ingress")

	vz := ctx.EffectiveCR()
	rancherHostName, err := getRancherHostname(c, vz)
	if err != nil {
		return log.ErrorfThrottledNewErr("Failed getting Rancher hostname: %s", err.Error())
	}

	if err := putServerURL(c, fmt.Sprintf("https://%s", rancherHostName)); err != nil {
		return log.ErrorfThrottledNewErr("Failed setting Rancher server URL: %s", err.Error())
	}

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

	if err := removeBootstrapSecretIfExists(log, c); err != nil {
		return log.ErrorfThrottledNewErr("Failed removing Rancher bootstrap secret: %s", err.Error())
	}

	if err := configureUISettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher UI settings: %s", err.Error())
	}
	// Create Fluentbit filter and parser for Rancher in cattle-fleet-system namespace
	if err := common.CreateOrDeleteFluentbitFilterAndParser(ctx, fluentbitFilterAndParserTemplate, FleetSystemNamespace, false); err != nil {
		return err
	}
	if err := r.HelmComponent.PostInstall(ctx); err != nil {
		return err
	}

	dynClient, err := getDynamicClientFunc()()
	if err != nil {
		return err
	}
	if err = common.UpdateKontainerDriverURLs(ctx, dynClient); err != nil {
		return err
	}
	return activateKontainerDrivers(ctx, dynClient)
}

// PreUninstall - prepare for Rancher uninstall
func (r rancherComponent) PreUninstall(ctx spi.ComponentContext) error {
	return preUninstall(ctx, r.monitor)
}

// PostUninstall handles the deletion of all Rancher resources after the Helm uninstall
func (r rancherComponent) PostUninstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("Rancher postUninstall dry run")
		return nil
	}
	// Delete Fluentbit filter and parser for Rancher in cattle-fleet-system namespace
	if err := common.CreateOrDeleteFluentbitFilterAndParser(ctx, fluentbitFilterAndParserTemplate, FleetSystemNamespace, true); err != nil {
		return err
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

	if err := configureUISettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring rancher UI settings: %s", err.Error())
	}

	if err := r.HelmComponent.PostUpgrade(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed helm component post upgrade: %s", err.Error())
	}

	if err := patchRancherIngress(c, ctx.EffectiveCR()); err != nil {
		return err
	}

	dynClient, err := getDynamicClientFunc()()
	if err != nil {
		return err
	}
	if err = common.UpdateKontainerDriverURLs(ctx, dynClient); err != nil {
		return err
	}
	if err := activateKontainerDrivers(ctx, dynClient); err != nil {
		return log.ErrorfThrottledNewErr("Failed to activate kontainerdriver post upgrade: %s", err.Error())
	}
	return cleanupRancherResources(context.TODO(), ctx.Client())
}

// Reconcile for the Rancher component
func (r rancherComponent) Reconcile(ctx spi.ComponentContext) error {
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

		if err := configureAuthSettings(ctx); err != nil {
			return ctx.Log().ErrorfThrottledNewErr("failed configuring rancher auth settings: %s", err.Error())
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

	if err := createOrUpdateUILogoSetting(ctx, SettingUILogoLight, SettingUILogoLightFile); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring %s setting for logo %s: %s", SettingUILogoLight, SettingUILogoLightFile, err.Error())
	}

	if err := createOrUpdateUILogoSetting(ctx, SettingUILogoDark, SettingUILogoDarkFile); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring %s setting for logo path %s: %s", SettingUILogoDark, SettingUILogoDarkFile, err.Error())
	}

	if err := createOrUpdateUIColorSettings(ctx); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui color settings: %s", err.Error())
	}

	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingUIBrand}, common.GVKSetting, map[string]interface{}{"value": SettingUIBrandValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring ui-brand setting: %s", err.Error())
	}

	return nil
}

// configureAuthSettings configures Rancher auth settings required for the Rancher Keycloak auth integration.
func configureAuthSettings(ctx spi.ComponentContext) error {
	log := ctx.Log()
	// Set "auth-user-info-resync-cron" to run the resync cron once in 15 minutes, less than the Keycloak default
	// SSO idle timeout (30 minutes)
	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingAuthResyncCron}, common.GVKSetting,
		map[string]interface{}{"value": SettingAuthResyncCronValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring auth-user-info-resync-cron setting: %s",
			err.Error())
	}

	// Set "auth-user-info-max-age-seconds" to "600", less than the Keycloak default SSO idle timeout (1800 seconds)
	// and less than the interval set for auth resync cron
	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingAuthMaxAge}, common.GVKSetting,
		map[string]interface{}{"value": SettingAuthMaxAgeValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring auth-user-info-max-age-seconds setting: %s",
			err.Error())
	}

	// Set "auth-user-session-ttl-minutes" to "540", less than the Keycloak default SSO session max (600 minutes)
	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingAuthTTL}, common.GVKSetting,
		map[string]interface{}{"value": SettingAuthTTLValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring auth-user-session-ttl-minutes setting: %s",
			err.Error())
	}

	// Set "kubeconfig-default-token-ttl-minutes" to "540", less than the Keycloak default SSO session max (600 minutes)
	if err := createOrUpdateResource(ctx, types.NamespacedName{Name: SettingKubeDefaultTokenTTL}, common.GVKSetting,
		map[string]interface{}{"value": SettingKubeDefaultTokenTTLValue}); err != nil {
		return log.ErrorfThrottledNewErr("failed configuring kubeconfig-default-token-ttl-minutes setting: %s",
			err.Error())
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

// IsClusterProvisionedByOCNEContainerDriver checks if the Kubernetes cluster was provisioned by the Rancher OCNE container driver.
func IsClusterProvisionedByOCNEContainerDriver() (bool, error) {
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return false, err
	}
	dynClient, err := k8sutil.GetDynamicClientFunc()
	if err != nil {
		return false, err
	}

	return checkContainerDriverProvisionedFunc(client, dynClient)
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
				if kerrs.IsNotFound(err) {
					return false, nil
				}
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

// checkContainerDriverProvisioned checks if the Kubernetes cluster was provisioned by the OCNE KontainerDriver.
func checkContainerDriverProvisioned(client corev1.CoreV1Interface, dynClient dynamic.Interface) (bool, error) {
	// Find the provisioner configmap resource and check if ociocne is indicated as the driver.
	_, err := client.ConfigMaps(constants.DefaultNamespace).Get(context.TODO(), clusterProvisioner, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// NOTE: not checking driver type since the existence of configmap alone indicates a container driver is responsible
	// for the provisioning of the cluster
	return true, nil
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

// isRancherDeploymentUpdateInProgress Checks only the cattle-system/rancher deployment is in progress or not
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
	currentCABundleInSecret, found, err := r.getCurrentCABundleSecretsValue(ctx, rancherTLSCASecretName, caCertsPem)
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

// restartRancherDeployment Performs a rolling restart of the Rancher deployment
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

// activateKontainerDrivers - activate OCI OCNE and OKE CAPI kontainerdrivers
func activateKontainerDrivers(ctx spi.ComponentContext, dynClient dynamic.Interface) error {
	for _, name := range []string{common.KontainerDriverOCIName, common.KontainerDriverOKECAPIName} {
		if err := common.ActivateKontainerDriver(ctx, dynClient, name); err != nil {
			return err
		}
	}
	return nil
}
