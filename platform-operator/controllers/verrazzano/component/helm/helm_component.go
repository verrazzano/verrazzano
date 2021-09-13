// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/secret"
	"io/ioutil"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const vzDefaultNamespace = constants.VerrazzanoSystemNamespace

// HelmComponent struct needed to implement a component
type HelmComponent struct {
	// ReleaseName is the helm chart release name
	ReleaseName string

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

	// PreUpgradeFunc is an optional function to run before upgrading
	PreUpgradeFunc preUpgradeFuncSig

	// AppendOverridesFunc is an optional function get additional override values
	AppendOverridesFunc appendOverridesSig

	// ReadyStatusFunc is an optional function override to do deeper checks on a component's ready state
	ReadyStatusFunc readyStatusFuncSig

	// ResolveNamespaceFunc is an optional function to process the namespace name
	ResolveNamespaceFunc resolveNamespaceSig

	//SupportsOperatorInstall Indicates whether or not the component supports install via the operator
	SupportsOperatorInstall bool

	//WaitForInstall Indicates if the operator should wait for helm operationsto complete (synchronous behavior)
	WaitForInstall bool

	// ImagePullSecretKeyname is the Helm Value Key for the image pull secret for a chart
	ImagePullSecretKeyname string

	// Dependencies is a list of Dependencies for this component, by component/release name
	Dependencies []string
}

// Verify that HelmComponent implements Component
var _ spi.Component = HelmComponent{}

// preInstallFuncSig is the signature for the optional function to run before installing; any KeyValue pairs should be prepended to the Helm overrides list
type preInstallFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) ([]bom.KeyValue, error)

// preUpgradeFuncSig is the signature for the optional preUgrade function
type preUpgradeFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error

// appendOverridesSig is an optional function called to generate additional overrides.
type appendOverridesSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, kvs []bom.KeyValue) ([]bom.KeyValue, error)

// resolveNamespaceSig is an optional function called for special namespace processing
type resolveNamespaceSig func(ns string) string

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overrideFiles ...string) (stdout []byte, stderr []byte, err error)

// readyStatusFuncSig describes the function signature for doing deeper checks on a component's ready state
type readyStatusFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string) bool

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = helm.Upgrade

// UpgradePrehooksEnabled is needed so that higher level units tests can disable as needed
var UpgradePrehooksEnabled = true

// Name returns the component name
func (h HelmComponent) Name() string {
	return h.ReleaseName
}

// GetDependencies returns the Dependencies of this component
func (h HelmComponent) GetDependencies() []string {
	return h.Dependencies
}

// IsOperatorInstallSupported Returns true if the component supports direct install via the operator
func (h HelmComponent) IsOperatorInstallSupported() bool {
	return h.SupportsOperatorInstall
}

// IsInstalled Indicates whether or not the component is installed
func (h HelmComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, namespace string) bool {
	installed, _ := helm.IsReleaseInstalled(h.ReleaseName, resolveNamespace(h, namespace))
	return installed
}

// IsReady Indicates whether or not a component is available and ready
func (h HelmComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	ns := resolveNamespace(h, namespace)
	installed, _ := helm.IsReleaseInstalled(h.ReleaseName, resolveNamespace(h, namespace))
	if installed {
		if h.ReadyStatusFunc != nil {
			return h.ReadyStatusFunc(log, client, h.ReleaseName, ns)
		}
		return true
	}
	return false
}

func (h HelmComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {

	// Resolve the namespace
	resolvedNamespace := resolveNamespace(h, namespace)

	failed, err := helm.IsReleaseFailed(h.ReleaseName, resolvedNamespace)
	if err != nil {
		return err
	}
	if failed {
		// Chart install failed, reset the chart to start over
		// NOTE: we'll likely have to put in some more logic akin to what we do for the scripts, see
		//       reset_chart() in the common.sh script.  Recovering chart state can be a bit difficult, we
		//       may need to draw on both the 'ls' and 'status' output for that.
		helm.Uninstall(log, h.ReleaseName, resolvedNamespace, dryRun)
	}

	var kvs []bom.KeyValue
	if h.PreInstallFunc != nil {
		preInstallValues, err := h.PreInstallFunc(log, client, h.ReleaseName, resolvedNamespace, h.ChartDir)
		if err != nil {
			return err
		}
		kvs = append(kvs, preInstallValues...)
	}
	// check for global image pull secret
	kvs, err = secret.AddGlobalImagePullSecretHelmOverride(log, client, resolvedNamespace, kvs, h.ImagePullSecretKeyname)
	if err != nil {
		return err
	}

	// vz-specific chart overrides file
	overridesString, err := h.buildOverridesString(log, client, resolvedNamespace, kvs...)
	if err != nil {
		return err
	}

	// Perform a helm upgrade --install
	_, _, err = upgradeFunc(log, h.ReleaseName, resolvedNamespace, h.ChartDir, h.WaitForInstall, dryRun, overridesString, h.ValuesFile)
	return err
}

// Upgrade is done by using the helm chart upgrade command.  This command will apply the latest chart
// that is included in the operator image, while retaining any helm Value overrides that were applied during
// install. Along with the override files in helm_config, we need to generate image overrides using the
// BOM json file.  Each component also has the ability to add additional override parameters.
func (h HelmComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	// Resolve the namespace
	namespace := resolveNamespace(h, ns)

	// Check if the component is installed before trying to upgrade
	found, err := helm.IsReleaseInstalled(h.ReleaseName, namespace)
	if err != nil {
		return err
	}
	if !found {
		log.Infof("Skipping upgrade of component %s since it is not installed", h.ReleaseName)
		return nil
	}

	// Do the preUpgrade if the function is defined
	if h.PreUpgradeFunc != nil && UpgradePrehooksEnabled {
		log.Infof("Running preUpgrade function for %s", h.ReleaseName)
		err := h.PreUpgradeFunc(log, client, h.ReleaseName, namespace, h.ChartDir)
		if err != nil {
			return err
		}
	}

	overridesString, err := h.buildOverridesString(log, client, namespace)
	if err != nil {
		return err
	}

	stdout, err := helm.GetValues(log, h.ReleaseName, namespace)
	if err != nil {
		return err
	}

	var tmpFile *os.File
	tmpFile, err = ioutil.TempFile(os.TempDir(), "values-*.yaml")
	if err != nil {
		log.Errorf("Failed to create temporary file: %v", err)
		return err
	}

	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write(stdout); err != nil {
		log.Errorf("Failed to write to temporary file: %v", err)
		return err
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		log.Errorf("Failed to close temporary file: %v", err)
		return err
	}

	log.Infof("Created values file: %s", tmpFile.Name())

	// Perform a helm upgrade --install
	_, _, err = upgradeFunc(log, h.ReleaseName, namespace, h.ChartDir, true, dryRun, overridesString, h.ValuesFile, tmpFile.Name())
	return err
}

func (h HelmComponent) buildOverridesString(log *zap.SugaredLogger, _ clipkg.Client, namespace string, additionalValues ...bom.KeyValue) (string, error) {
	// Optionally create a second override file.  This will contain both image overridesString and any additional
	// overridesString required by a component.
	// Get image overridesString unless opt out
	var kvs []bom.KeyValue
	var err error
	if !h.IgnoreImageOverrides {
		kvs, err = getImageOverrides(h.ReleaseName)
		if err != nil {
			return "", err
		}
	}

	// Append any additional overridesString for the component (see Keycloak.go for example)
	if h.AppendOverridesFunc != nil {
		overrideValues, err := h.AppendOverridesFunc(log, h.ReleaseName, namespace, h.ChartDir, []bom.KeyValue{})
		if err != nil {
			return "", err
		}
		kvs = append(kvs, overrideValues...)
	}

	// Append any special overrides passed in
	if len(additionalValues) > 0 {
		kvs = append(kvs, additionalValues...)
	}

	// If there are overridesString the create a comma separated string
	var overridesString string
	if len(kvs) > 0 {
		bldr := strings.Builder{}
		for i, kv := range kvs {
			if i > 0 {
				bldr.WriteString(",")
			}
			bldr.WriteString(fmt.Sprintf("%s=%s", kv.Key, kv.Value))
		}
		overridesString = bldr.String()
	}
	return overridesString, nil
}

// resolveNamespace Resolve/normalize the namespace for a Helm-based component
//
// The need for this stems from an issue with the Verrazzano component and the fact
// that component charts underneath VZ component need to have the ns overridden
func resolveNamespace(h HelmComponent, ns string) string {
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

func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = helm.Upgrade
}
