// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"context"
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/helm"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
	corev1 "k8s.io/api/core/v1"
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
type getInstallOverridesSig func(context spi.ComponentContext) []vzapi.Overrides

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

// GetJsonName returns the josn name of the verrazzano component in CRD
func (h HelmComponent) GetJSONName() string {
	return h.JSONName
}

// GetOverrides returns the list of install overrides for a component
func (h HelmComponent) GetOverrides(ctx spi.ComponentContext) []vzapi.Overrides {
	if h.GetInstallOverridesFunc != nil {
		return h.GetInstallOverridesFunc(ctx)
	}
	return []vzapi.Overrides{}
}

// GetDependencies returns the Dependencies of this component
func (h HelmComponent) GetDependencies() []string {
	return h.Dependencies
}

// IsOperatorInstallSupported Returns true if the component supports direct install via the operator
func (h HelmComponent) IsOperatorInstallSupported() bool {
	return h.SupportsOperatorInstall
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
	installed, _ := helm.IsReleaseInstalled(h.ReleaseName, h.resolveNamespace(context.EffectiveCR().Namespace))
	return installed, nil
}

// IsReady Indicates whether a component is available and ready
func (h HelmComponent) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	// Does the Helm installed app_version number match the chart?
	chartInfo, err := helmcli.GetChartInfo(h.ChartDir)
	if err != nil {
		return false
	}
	releaseAppVersion, err := helmcli.GetReleaseAppVersion(h.ReleaseName, h.ChartNamespace)
	if err != nil {
		return false
	}
	if chartInfo.AppVersion != releaseAppVersion {
		return false
	}

	ns := h.resolveNamespace(context.EffectiveCR().Namespace)
	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, ns); !deployed {
		return false
	}

	return true
}

// IsEnabled Indicates whether a component is enabled for installation
func (h HelmComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	return true
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (h HelmComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (h HelmComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	return nil
}

func (h HelmComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	return true
}

// Install installs the component using Helm
func (h HelmComponent) Install(context spi.ComponentContext) error {

	// Resolve the namespace
	resolvedNamespace := h.resolveNamespace(context.EffectiveCR().Namespace)

	var kvs []bom.KeyValue
	// check for global image pull secret
	kvs, err := secret.AddGlobalImagePullSecretHelmOverride(context.Log(), context.Client(), resolvedNamespace, kvs, h.ImagePullSecretKeyname)
	if err != nil {
		return err
	}

	// vz-specific chart overrides file
	overrides, err := h.buildCustomHelmOverrides(context, resolvedNamespace, kvs...)
	defer vzos.RemoveTempFiles(context.Log().GetZapLogger(), `\w*`)
	if err != nil {
		return err
	}

	// Perform an install using the helm upgrade --install command
	_, _, err = upgradeFunc(context.Log(), h.ReleaseName, resolvedNamespace, h.ChartDir, h.WaitForInstall, context.IsDryRun(), overrides)
	return err
}

func (h HelmComponent) PreInstall(context spi.ComponentContext) error {
	if h.PreInstallFunc != nil {
		err := h.PreInstallFunc(context, h.ReleaseName, h.resolveNamespace(context.EffectiveCR().Namespace), h.ChartDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h HelmComponent) PostInstall(context spi.ComponentContext) error {
	if h.PostInstallFunc != nil {
		if err := h.PostInstallFunc(context, h.ReleaseName, h.resolveNamespace(context.EffectiveCR().Namespace)); err != nil {
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

// Upgrade is done by using the helm chart upgrade command.  This command will apply the latest chart
// that is included in the operator image, while retaining any helm Value overrides that were applied during
// install. Along with the override files in helm_config, we need to generate image overrides using the
// BOM json file.  Each component also has the ability to add additional override parameters.
func (h HelmComponent) Upgrade(context spi.ComponentContext) error {
	if h.SkipUpgrade {
		context.Log().Infof("Upgrade disabled for %s", h.ReleaseName)
		return nil
	}

	// Resolve the resolvedNamespace
	resolvedNamespace := h.resolveNamespace(context.EffectiveCR().Namespace)

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
	defer vzos.RemoveTempFiles(context.Log().GetZapLogger(), `\w*`)
	if err != nil {
		return err
	}

	stdout, err := helm.GetValues(context.Log(), h.ReleaseName, resolvedNamespace)
	if err != nil {
		return err
	}

	tmpFile, err := vzos.CreateTempFile(context.Log(), "values-*.yaml", stdout)
	if err != nil {
		return err
	}

	// Generate a list of override files making helm get values overrides first
	overrides = append([]helm.HelmOverrides{{FileOverride: tmpFile.Name()}}, overrides...)

	_, _, err = upgradeFunc(context.Log(), h.ReleaseName, resolvedNamespace, h.ChartDir, true, context.IsDryRun(), overrides)
	return err
}

func (h HelmComponent) PreUpgrade(_ spi.ComponentContext) error {
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
	// Getting user defined install overrides as the highest priority
	kvs, err = h.retrieveInstallOverrideResources(context, h.GetOverrides(context))
	if err != nil {
		return overrides, err
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
	file, err := vzos.CreateTempFile(context.Log(), "helm-overrides-*.yaml", []byte(fileString))
	if err != nil {
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

// retrieveInstallOverrideResources takes the list of Overrides and returns a list of key value pairs
func (h HelmComponent) retrieveInstallOverrideResources(ctx spi.ComponentContext, overrides []vzapi.Overrides) ([]bom.KeyValue, error) {
	var kvs []bom.KeyValue
	for _, override := range overrides {
		// Check if ConfigMapRef is populated and gather helm file
		if override.ConfigMapRef != nil {
			// Get the ConfigMap
			configMap := &corev1.ConfigMap{}
			selector := override.ConfigMapRef
			nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
			optional := selector.Optional
			err := ctx.Client().Get(context.TODO(), nsn, configMap)
			if err != nil {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Configmap %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
					return kvs, err
				}
				ctx.Log().Debugf("Optional Configmap %s from namespace %s not found", nsn.Name, nsn.Namespace)
				continue
			}

			tmpFile, err := createInstallOverrideFile(ctx, nsn, configMap.Data, selector.Key, selector.Optional)
			if err != nil {
				return kvs, err
			}
			if tmpFile != nil {
				kvs = append(kvs, bom.KeyValue{Value: tmpFile.Name(), IsFile: true})
			}
		}
		// Check if SecretRef is populated and gather helm file
		if override.SecretRef != nil {
			// Get the Secret
			sec := &corev1.Secret{}
			selector := override.SecretRef
			nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
			optional := selector.Optional
			err := ctx.Client().Get(context.TODO(), nsn, sec)
			if err != nil {
				if optional == nil || !*optional {
					err := ctx.Log().ErrorfNewErr("Could not get Secret %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
					return kvs, err
				}
				ctx.Log().Debugf("Optional Secret %s from namespace %s not found", nsn.Name, nsn.Namespace)
				continue
			}

			dataStrings := map[string]string{}
			for key, val := range sec.Data {
				dataStrings[key] = string(val)
			}
			tmpFile, err := createInstallOverrideFile(ctx, nsn, dataStrings, selector.Key, selector.Optional)
			if err != nil {
				return kvs, err
			}
			if tmpFile != nil {
				kvs = append(kvs, bom.KeyValue{Value: tmpFile.Name(), IsFile: true})
			}
		}
	}
	return kvs, nil
}

// createInstallOverrideFile takes in the data from a kubernetes resource and creates a temporary file for helm install
func createInstallOverrideFile(ctx spi.ComponentContext, nsn types.NamespacedName, data map[string]string, dataKey string, optional *bool) (*os.File, error) {
	var file *os.File

	// Get resource data
	fieldData, ok := data[dataKey]
	if !ok {
		if optional == nil || !*optional {
			err := ctx.Log().ErrorfNewErr("Could not get Data field %s from Resource %s from namespace %s", dataKey, nsn.Name, nsn.Namespace)
			return file, err
		}
		ctx.Log().Debugf("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, dataKey)
		return file, nil
	}

	// Create the temp file for the data
	file, err := vzos.CreateTempFile(ctx.Log(), "helm-overrides-*.yaml", []byte(fieldData))
	if err != nil {
		return file, err
	}
	return file, nil
}

// resolveNamespace Resolve/normalize the namespace for a Helm-based component
//
// The need for this stems from an issue with the Verrazzano component and the fact
// that component charts underneath VZ component need to have the ns overridden
func (h HelmComponent) resolveNamespace(ns string) string {
	namespace := ns
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
func GetInstallArgs(args []vzapi.InstallArgs) []bom.KeyValue {
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
