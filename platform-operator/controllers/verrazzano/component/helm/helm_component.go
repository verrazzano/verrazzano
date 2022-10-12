// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	ctx "context"
	"fmt"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"
	"os"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmComponent struct needed to implement a component
type HelmComponent struct {
	// ReleaseName is the helm chart release name
	ReleaseName string

	// JSONName is the josn name of the verrazzano component in CRD
	JSONName string

	// ChartDir is the helm chart directory
	ChartDir string

	// ChartNamespace is the namespace passed to the helm command
	ChartNamespace string

	// IgnoreNamespaceOverride bool indicates that the namespace param passed to
	// Upgrade is ignored
	IgnoreNamespaceOverride bool

	// IgnoreImageOverrides bool indicates that the image overrides processing should be ignored
	// This should only be set to true if the component doesn't have images (like istio-base) in
	// which case it is not in the bom
	IgnoreImageOverrides bool

	// ValuesFile is the helm chart values override file
	ValuesFile string

	// InstallBeforeUpgrade if component can be installed before upgade is done, default false
	InstallBeforeUpgrade bool

	// PreInstallFunc is an optional function to run before installing
	PreInstallFunc preInstallFuncSig

	// PostInstallFunc is an optional function to run after installing
	PostInstallFunc postInstallFuncSig

	// PreUpgradeFunc is an optional function to run before upgrading
	PreUpgradeFunc preUpgradeFuncSig

	// AppendOverridesFunc is an optional function get additional override values
	AppendOverridesFunc appendOverridesSig

	// GetInstallOverridesFunc is an optional function get install override sources
	GetInstallOverridesFunc getInstallOverridesSig

	// ResolveNamespaceFunc is an optional function to process the namespace name
	ResolveNamespaceFunc resolveNamespaceSig

	// SupportsOperatorInstall Indicates whether or not the component supports install via the operator
	SupportsOperatorInstall bool

	// SupportsOperatorUninstall Indicates whether or not the component supports uninstall via the operator
	SupportsOperatorUninstall bool

	// WaitForInstall Indicates if the operator should wait for helm operations to complete (synchronous behavior)
	WaitForInstall bool

	// ImagePullSecretKeyname is the Helm Value Key for the image pull secret for a chart
	ImagePullSecretKeyname string

	// Dependencies is a list of Dependencies for this component, by component/release name
	Dependencies []string

	// SkipUpgrade when true will skip upgrading this component in the upgrade loop
	// This is for the istio helm components
	SkipUpgrade bool

	// The minimum required Verrazzano version.
	MinVerrazzanoVersion string

	// Ingress names associated with the component
	IngressNames []types.NamespacedName

	// Certificates associated with the component
	Certificates []types.NamespacedName
}

// Verify that HelmComponent implements Component
var _ spi.Component = HelmComponent{}

// preInstallFuncSig is the signature for the optional function to run before installing; any KeyValue pairs should be prepended to the Helm overrides list
type preInstallFuncSig func(context spi.ComponentContext, releaseName string, namespace string, chartDir string) error

// postInstallFuncSig is the signature for the optional function to run before installing; any KeyValue pairs should be prepended to the Helm overrides list
type postInstallFuncSig func(context spi.ComponentContext, releaseName string, namespace string) error

// preUpgradeFuncSig is the signature for the optional preUgrade function
type preUpgradeFuncSig func(log vzlog.VerrazzanoLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error

// appendOverridesSig is an optional function called to generate additional overrides.
type appendOverridesSig func(context spi.ComponentContext, releaseName string, namespace string, chartDir string, kvs []bom.KeyValue) ([]bom.KeyValue, error)

// getInstallOverridesSig is an optional function called to generate additional overrides.
type getInstallOverridesSig func(object runtime.Object) interface{}

// resolveNamespaceSig is an optional function called for special namespace processing
type resolveNamespaceSig func(ns string) string

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helm.HelmOverrides) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = helm.Upgrade

func SetUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func SetDefaultUpgradeFunc() {
	upgradeFunc = helm.Upgrade
}

// UpgradePrehooksEnabled is needed so that higher level units tests can disable as needed
var UpgradePrehooksEnabled = true

// Name returns the component name
func (h HelmComponent) Name() string {
	return h.ReleaseName
}

// Namespace returns the component namespace
func (h HelmComponent) Namespace() string {
	return h.ChartNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done
func (h HelmComponent) ShouldInstallBeforeUpgrade() bool {
	return h.InstallBeforeUpgrade
}

// GetJsonName returns the josn name of the verrazzano component in CRD
func (h HelmComponent) GetJSONName() string {
	return h.JSONName
}

// GetOverrides returns the list of install overrides for a component
func (h HelmComponent) GetOverrides(cr runtime.Object) interface{} {
	if h.GetInstallOverridesFunc != nil {
		return h.GetInstallOverridesFunc(cr)
	}
	if _, ok := cr.(*v1beta1.Verrazzano); ok {
		return []v1beta1.Overrides{}
	}
	return []v1alpha1.Overrides{}

}

// GetDependencies returns the Dependencies of this component
func (h HelmComponent) GetDependencies() []string {
	return h.Dependencies
}

// IsOperatorInstallSupported Returns true if the component supports direct install via the operator
func (h HelmComponent) IsOperatorInstallSupported() bool {
	return h.SupportsOperatorInstall
}

// IsOperatorUninstallSupported Returns true if the component supports direct uninstall via the operator
func (h HelmComponent) IsOperatorUninstallSupported() bool {
	return h.SupportsOperatorUninstall
}

// GetCertificateNames returns the list of expected certificates used by this component
func (h HelmComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return h.Certificates
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by this component
func (h HelmComponent) GetMinVerrazzanoVersion() string {
	if len(h.MinVerrazzanoVersion) == 0 {
		return constants.VerrazzanoVersion1_0_0
	}
	return h.MinVerrazzanoVersion
}

// IsInstalled Indicates whether or not the component is installed
func (h HelmComponent) IsInstalled(context spi.ComponentContext) (bool, error) {
	if context.IsDryRun() {
		context.Log().Debugf("IsInstalled() dry run for %s", h.ReleaseName)
		return true, nil
	}
	installed, _ := helm.IsReleaseInstalled(h.ReleaseName, h.resolveNamespace(context))
	return installed, nil
}

// IsReady Indicates whether a component is available and ready
func (h HelmComponent) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	// Does the Helm installed app_version number match the chart?
	chartInfo, err := helm.GetChartInfo(h.ChartDir)
	if err != nil {
		return false
	}
	releaseAppVersion, err := helm.GetReleaseAppVersion(h.ReleaseName, h.ChartNamespace)
	if err != nil {
		return false
	}
	if chartInfo.AppVersion != releaseAppVersion {
		return false
	}

	ns := h.resolveNamespace(context)
	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, ns); deployed {
		return true
	}
	return false
}

// IsEnabled Indicates whether a component is enabled for installation
func (h HelmComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return true
}

func (h HelmComponent) v1alpha1Validate(vz *v1alpha1.Verrazzano) error {
	if err := v1alpha1.ValidateInstallOverrides(h.GetOverrides(vz).([]v1alpha1.Overrides)); err != nil {
		return err
	}
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (h HelmComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	return h.v1alpha1Validate(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (h HelmComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	return h.v1alpha1Validate(new)
}

func (h HelmComponent) v1beta1Validate(vz *v1beta1.Verrazzano) error {
	if err := v1alpha1.ValidateInstallOverridesV1Beta1(h.GetOverrides(vz).([]v1beta1.Overrides)); err != nil {
		return err
	}
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (h HelmComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return h.v1beta1Validate(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (h HelmComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	return h.v1beta1Validate(new)
}

func (h HelmComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	return true
}

// Install installs the component using Helm
func (h HelmComponent) Install(context spi.ComponentContext) error {

	// Resolve the namespace
	resolvedNamespace := h.resolveNamespace(context)

	var kvs []bom.KeyValue
	// check for global image pull secret
	kvs, err := secret.AddGlobalImagePullSecretHelmOverride(context.Log(), context.Client(), resolvedNamespace, kvs, h.ImagePullSecretKeyname)
	if err != nil {
		return err
	}

	// vz-specific chart overrides file
	overrides, err := h.buildCustomHelmOverrides(context, resolvedNamespace, kvs...)
	defer vzos.RemoveTempFiles(context.Log().GetZapLogger(), `helm-overrides.*\.yaml`)
	if err != nil {
		return err
	}

	// Perform an install using the helm upgrade --install command
	_, _, err = upgradeFunc(context.Log(), h.ReleaseName, resolvedNamespace, h.ChartDir, h.WaitForInstall, context.IsDryRun(), overrides)
	return err
}

func (h HelmComponent) PreInstall(context spi.ComponentContext) error {
	releaseStatus, err := helm.GetReleaseStatus(context.Log(), h.ReleaseName, h.ChartNamespace)
	if err != nil {
		context.Log().ErrorfThrottledNewErr("Error getting release status for %s", h.ReleaseName)
	} else if releaseStatus != release.StatusDeployed.String() || releaseStatus == release.StatusUninstalled.String() { // When helm release is not deployed or uninstalled, cleanup the secret
		cleanupLatestSecret(context, h, true)
	}

	if h.PreInstallFunc != nil {
		err := h.PreInstallFunc(context, h.ReleaseName, h.resolveNamespace(context), h.ChartDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// cleanupLatestSecret is to cleanup the secrets to get helm installation going
// This deletes the latest secret that matches the release if the helm release status is not deployed or uninstalled
// If this is not vz install, do not delete the secret revision 1
func cleanupLatestSecret(context spi.ComponentContext, h HelmComponent, isInstall bool) {
	secretList := &v1.SecretList{}
	context.Client().List(ctx.TODO(), secretList, &clipkg.ListOptions{
		Namespace: h.ChartNamespace,
	})

	filteredHelmSecrets := []v1.Secret{}
	for _, eachSecret := range secretList.Items {
		if eachSecret.Type == "helm.sh/release.v1" && strings.Contains(eachSecret.Name, "sh.helm.release.v1."+h.ReleaseName+".") { // Filter only helm release type secrets and matching releaseName
			filteredHelmSecrets = append(filteredHelmSecrets, eachSecret)
		}
	}

	// No secrets matches found, so return
	if len(filteredHelmSecrets) == 0 {
		return
	}

	// Sort the secrets based on CreationTimeStamp; latest ones first
	sort.Slice(filteredHelmSecrets, func(i, j int) bool {
		return (filteredHelmSecrets[i].CreationTimestamp.Time).After(filteredHelmSecrets[j].CreationTimestamp.Time)
	})

	// When there is only one secret AND if it's not preinstall then skip
	if len(filteredHelmSecrets) == 1 && !isInstall {
		return
	}

	context.Log().Progressf("Deleting secret %s", filteredHelmSecrets[0])
	if err := context.Client().Delete(ctx.TODO(), &filteredHelmSecrets[0]); err != nil {
		context.Log().Errorf("Error deleting secret %s", filteredHelmSecrets[0])
	}
}

func (h HelmComponent) PostInstall(context spi.ComponentContext) error {
	if h.PostInstallFunc != nil {
		if err := h.PostInstallFunc(context, h.ReleaseName, h.resolveNamespace(context)); err != nil {
			return err
		}
	}

	// If the component has any ingresses associated, those should be present
	prefix := fmt.Sprintf("Component %s", h.Name())
	if !status.IngressesPresent(context.Log(), context.Client(), h.GetIngressNames(context), prefix) {
		return ctrlerrors.RetryableError{
			Source:    h.ReleaseName,
			Operation: "Check if Ingresses are present",
		}
	}

	if readyStatus, certsNotReady := status.CertificatesAreReady(context.Client(), context.Log(), context.EffectiveCR(), h.Certificates); !readyStatus {
		context.Log().Progressf("Certificates not ready for component %s: %v", h.ReleaseName, certsNotReady)
		return ctrlerrors.RetryableError{
			Source:    h.ReleaseName,
			Operation: "Check if certificates are ready",
		}
	}

	return nil
}

func (h HelmComponent) PreUninstall(context spi.ComponentContext) error {
	releaseStatus, err := helm.GetReleaseStatus(context.Log(), h.ReleaseName, h.ChartNamespace)
	if err != nil {
		context.Log().ErrorfThrottledNewErr("Error getting release status for %s", h.ReleaseName)
	}
	if releaseStatus == release.StatusDeployed.String() || releaseStatus == release.StatusUninstalled.String() { // When helm release is not deployed or uninstalled, cleanup the secret
		return nil
	}
	cleanupLatestSecret(context, h, false)
	return nil
}

func (h HelmComponent) Uninstall(context spi.ComponentContext) error {
	installed, err := h.IsInstalled(context)
	if err != nil {
		return err
	}
	if !installed {
		context.Log().Infof("%s already uninstalled", h.Name())
		return nil
	}
	_, stderr, err := helm.Uninstall(context.Log(), h.ReleaseName, h.resolveNamespace(context), context.IsDryRun())
	if err != nil {
		context.Log().Errorf("Error uninstalling %s, error: %s, stderr: %s", h.Name(), err.Error(), stderr)
		return err
	}
	return nil
}

func (h HelmComponent) PostUninstall(context spi.ComponentContext) error {
	return nil
}

// Upgrade is done by using the helm chart upgrade command.  This command will apply the latest chart
// that is included in the operator image, while retaining any helm Value overrides that were applied during
// install. Along with the override files in helm_config, we need to generate image overrides using the
// BOM json file.  Each component also has the ability to add additional override parameters.
func (h HelmComponent) Upgrade(context spi.ComponentContext) error {
	if h.SkipUpgrade {
		context.Log().Infof("Upgrade disabled for %s", h.ReleaseName)
		return nil
	}

	// Resolve the namespace
	resolvedNamespace := h.resolveNamespace(context)

	// Check if the component is installed before trying to upgrade
	found, err := helm.IsReleaseInstalled(h.ReleaseName, resolvedNamespace)
	if err != nil {
		return err
	}
	if !found {
		context.Log().Infof("Skipping upgrade of component %s since it is not installed", h.ReleaseName)
		return nil
	}

	// Do the preUpgrade if the function is defined
	if h.PreUpgradeFunc != nil && UpgradePrehooksEnabled {
		context.Log().Infof("Running preUpgrade function for %s", h.ReleaseName)
		err := h.PreUpgradeFunc(context.Log(), context.Client(), h.ReleaseName, resolvedNamespace, h.ChartDir)
		if err != nil {
			return err
		}
	}

	// check for global image pull secret
	var kvs []bom.KeyValue
	kvs, err = secret.AddGlobalImagePullSecretHelmOverride(context.Log(), context.Client(), resolvedNamespace, kvs, h.ImagePullSecretKeyname)
	if err != nil {
		return err
	}

	overrides, err := h.buildCustomHelmOverrides(context, resolvedNamespace, kvs...)
	defer vzos.RemoveTempFiles(context.Log().GetZapLogger(), `helm-overrides.*\.yaml`)
	if err != nil {
		return err
	}

	stdout, err := helm.GetValues(context.Log(), h.ReleaseName, resolvedNamespace)
	if err != nil {
		return err
	}

	tmpFile, err := vzos.CreateTempFile("helm-overrides-values-*.yaml", stdout)
	if err != nil {
		context.Log().Error(err.Error())
		return err
	}

	// Generate a list of override files making helm get values overrides first
	overrides = append([]helm.HelmOverrides{{FileOverride: tmpFile.Name()}}, overrides...)

	_, _, err = upgradeFunc(context.Log(), h.ReleaseName, resolvedNamespace, h.ChartDir, true, context.IsDryRun(), overrides)
	return err
}

func (h HelmComponent) PreUpgrade(context spi.ComponentContext) error {
	releaseStatus, err := helm.GetReleaseStatus(context.Log(), h.ReleaseName, h.ChartNamespace)
	if err != nil {
		context.Log().ErrorfThrottledNewErr("Error getting release status for %s", h.ReleaseName)
		return err
	}
	if releaseStatus == release.StatusDeployed.String() || releaseStatus == release.StatusUninstalled.String() { // When helm release is deployed or not uninstalled, skip
		return nil
	}
	cleanupLatestSecret(context, h, false)
	return nil
}

func (h HelmComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (h HelmComponent) Reconcile(_ spi.ComponentContext) error {
	return nil
}

// buildCustomHelmOverrides Builds the helm overrides for a release, including image and file, and custom overrides
// - returns an error and a HelmOverride struct with the field populated
func (h HelmComponent) buildCustomHelmOverrides(context spi.ComponentContext, namespace string, additionalValues ...bom.KeyValue) ([]helm.HelmOverrides, error) {
	// Optionally create a second override file.  This will contain both image setOverrides and any additional
	// setOverrides required by a component.
	// Get image setOverrides unless opt out
	var kvs []bom.KeyValue
	var err error
	var overrides []helm.HelmOverrides

	// Sort the kvs list by priority (0th term has the highest priority)

	// Getting user defined Helm overrides as the highest priority
	overrideStrings, err := common.GetInstallOverridesYAML(context, h.GetOverrides(context.EffectiveCR()).([]v1alpha1.Overrides))
	if err != nil {
		return overrides, err
	}
	for _, overrideString := range overrideStrings {
		file, err := vzos.CreateTempFile(fmt.Sprintf("helm-overrides-user-%s-*.yaml", h.Name()), []byte(overrideString))
		if err != nil {
			context.Log().Error(err.Error())
			return overrides, err
		}
		kvs = append(kvs, bom.KeyValue{Value: file.Name(), IsFile: true})
	}

	// Create files from the Verrazzano Helm values
	newKvs, err := h.filesFromVerrazzanoHelm(context, namespace, additionalValues)
	if err != nil {
		return overrides, err
	}
	kvs = append(kvs, newKvs...)

	// Add the values file ot the file overrides
	if len(h.ValuesFile) > 0 {
		kvs = append(kvs, bom.KeyValue{Value: h.ValuesFile, IsFile: true})
	}

	// Convert the key value pairs to Helm overrides
	overrides = h.organizeHelmOverrides(kvs)
	return overrides, nil
}

func (h HelmComponent) filesFromVerrazzanoHelm(context spi.ComponentContext, namespace string, additionalValues []bom.KeyValue) ([]bom.KeyValue, error) {
	var kvs []bom.KeyValue
	var newKvs []bom.KeyValue

	// Get image overrides if they are specified
	if !h.IgnoreImageOverrides {
		imageOverrides, err := getImageOverrides(h.ReleaseName)
		if err != nil {
			return newKvs, err
		}
		kvs = append(kvs, imageOverrides...)
	}

	// Append any additional setOverrides for the component (see Keycloak.go for example)
	if h.AppendOverridesFunc != nil {
		overrideValues, err := h.AppendOverridesFunc(context, h.ReleaseName, namespace, h.ChartDir, []bom.KeyValue{})
		if err != nil {
			return newKvs, err
		}
		kvs = append(kvs, overrideValues...)
	}

	// Append any special overrides passed in
	if len(additionalValues) > 0 {
		kvs = append(kvs, additionalValues...)
	}

	// Expand the existing kvs values into expected format
	var fileValues []bom.KeyValue
	for _, kv := range kvs {
		// If the value is a file, add it to the new kvs
		if kv.IsFile {
			newKvs = append(newKvs, kv)
			continue
		}

		// If set file, extract the data into the value parameter
		if kv.SetFile {
			data, err := os.ReadFile(kv.Value)
			if err != nil {
				return newKvs, context.Log().ErrorfNewErr("Could not open file %s: %v", kv.Value, err)
			}
			kv.Value = string(data)
		}

		fileValues = append(fileValues, kv)
	}

	// Take the YAML values and construct a YAML file
	// This uses the Helm YAML formatting
	fileString, err := yaml.HelmValueFileConstructor(fileValues)
	if err != nil {
		return newKvs, context.Log().ErrorfNewErr("Could not create YAML file from key value pairs: %v", err)
	}

	// Create the file from the string
	file, err := vzos.CreateTempFile("helm-overrides-verrazzano-*.yaml", []byte(fileString))
	if err != nil {
		context.Log().Error(err.Error())
		return newKvs, err
	}
	if file != nil {
		newKvs = append(newKvs, bom.KeyValue{Value: file.Name(), IsFile: true})
	}
	return newKvs, nil
}

// organizeHelmOverrides creates a list of Helm overrides from key value pairs in reverse precedence (0th value has the lowest precedence)
// Each key value pair gets its own override object to keep strict precedence
func (h HelmComponent) organizeHelmOverrides(kvs []bom.KeyValue) []helm.HelmOverrides {
	var overrides []helm.HelmOverrides
	for _, kv := range kvs {
		if kv.SetString {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetStringOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		} else if kv.SetFile {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetFileOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		} else if kv.IsFile {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{FileOverride: kv.Value}}, overrides...)
		} else {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		}
	}
	return overrides
}

// resolveNamespace Resolve/normalize the namespace for a Helm-based component
//
// The need for this stems from an issue with the Verrazzano component and the fact
// that component charts underneath VZ component need to have the ns overridden
func (h HelmComponent) resolveNamespace(ctx spi.ComponentContext) string {
	namespace := ctx.EffectiveCR().Namespace
	if h.ResolveNamespaceFunc != nil {
		namespace = h.ResolveNamespaceFunc(namespace)
	}
	if h.IgnoreNamespaceOverride {
		namespace = h.ChartNamespace
	}
	return namespace
}

// Get the image overrides from the BOM
func getImageOverrides(subcomponentName string) ([]bom.KeyValue, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	numImages := bomFile.GetSubcomponentImageCount(subcomponentName)
	if numImages == 0 {
		return []bom.KeyValue{}, nil
	}

	kvs, err := bomFile.BuildImageOverrides(subcomponentName)
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

func (h HelmComponent) GetSkipUpgrade() bool {
	return h.SkipUpgrade
}

func (h HelmComponent) GetIngressNames(context spi.ComponentContext) []types.NamespacedName {
	return h.IngressNames
}

// GetInstallArgs returns the list of install args as Helm value pairs
func GetInstallArgs(args []v1alpha1.InstallArgs) []bom.KeyValue {
	installArgs := []bom.KeyValue{}
	for _, arg := range args {
		installArg := bom.KeyValue{}
		if arg.Value != "" {
			installArg.Key = arg.Name
			installArg.Value = arg.Value
			if arg.SetString {
				installArg.SetString = arg.SetString
			}
			installArgs = append(installArgs, installArg)
			continue
		}
		for i, value := range arg.ValueList {
			installArg.Key = fmt.Sprintf("%s[%d]", arg.Name, i)
			installArg.Value = value
			installArgs = append(installArgs, installArg)
		}
	}
	return installArgs
}
