// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const vzDefaultNamespace = constants.VerrazzanoSystemNamespace

// helmComponent struct needed to implement a component
type helmComponent struct {
	// releaseName is the helm chart release name
	releaseName string

	// chartDir is the helm chart directory
	chartDir string

	// chartNamespace is the namespace passed to the helm command
	chartNamespace string

	// ignoreNamespaceOverride bool indicates that the namespace param passed to
	// Upgrade is ignored
	ignoreNamespaceOverride bool

	// ignoreImageOverrides bool indicates that the image overrides processing should be ignored
	// This should only be set to true if the component doesn't have images (like istio-base) in
	// which case it is not in the bom
	ignoreImageOverrides bool

	// valuesFile is the helm chart values override file
	valuesFile string

	// preInstallFunc is an optional function to run before installing
	preInstallFunc preInstallFuncSig

	// preUpgradeFunc is an optional function to run before upgrading
	preUpgradeFunc preUpgradeFuncSig

	// appendOverridesFunc is an optional function get additional override values
	appendOverridesFunc appendOverridesSig

	// readyStatusFunc is an optional function override to do deeper checks on a component's ready state
	readyStatusFunc readyStatusFuncSig

	// resolveNamespaceFunc is an optional function to process the namespace name
	resolveNamespaceFunc resolveNamespaceSig

	//supportsInstall Indicates whether or not the component supports install via the operator
	supportsOperatorInstall bool

	//waitForInstall Indicates if the operator should wait for helm operationsto complete (synchronous behavior)
	waitForInstall bool

	// imagePullSecretKeyname is the Helm value key for the image pull secret for a chart
	imagePullSecretKeyname string

	// dependencies is a list of dependencies for this component, by component/release name
	dependencies []string
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

// preInstallFuncSig is the signature for the optional function to run before installing; any keyValue pairs should be prepended to the Helm overrides list
type preInstallFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) ([]keyValue, error)

// preUpgradeFuncSig is the signature for the optional preUgrade function
type preUpgradeFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error

// appendOverridesSig is an optional function called to generate additional overrides.
type appendOverridesSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, kvs []keyValue) ([]keyValue, error)

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
func (h helmComponent) Name() string {
	return h.releaseName
}

// GetDependencies returns the dependencies of this component
func (h helmComponent) GetDependencies() []string {
	return h.dependencies
}

// IsOperatorInstallSupported Returns true if the component supports direct install via the operator
func (h helmComponent) IsOperatorInstallSupported() bool {
	return h.supportsOperatorInstall
}

// IsInstalled Indicates whether or not the component is installed
func (h helmComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, namespace string) bool {
	installed, _ := helm.IsReleaseInstalled(h.releaseName, resolveNamespace(h, namespace))
	return installed
}

// IsReady Indicates whether or not a component is available and ready
func (h helmComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	ns := resolveNamespace(h, namespace)
	installed, _ := helm.IsReleaseInstalled(h.releaseName, resolveNamespace(h, namespace))
	if installed {
		if h.readyStatusFunc != nil {
			return h.readyStatusFunc(log, client, h.releaseName, ns)
		}
		return true
	}
	return false
}

func (h helmComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {

	// Resolve the namespace
	resolvedNamespace := resolveNamespace(h, namespace)

	failed, err := helm.IsReleaseFailed(h.releaseName, resolvedNamespace)
	if err != nil {
		return err
	}
	if failed {
		// Chart install failed, reset the chart to start over
		// NOTE: we'll likely have to put in some more logic akin to what we do for the scripts, see
		//       reset_chart() in the common.sh script.  Recovering chart state can be a bit difficult, we
		//       may need to draw on both the 'ls' and 'status' output for that.
		helm.Uninstall(log, h.releaseName, resolvedNamespace, h.waitForInstall, dryRun)
	}

	var kvs []keyValue
	if h.preInstallFunc != nil {
		preInstallValues, err := h.preInstallFunc(log, client, h.releaseName, resolvedNamespace, h.chartDir)
		if err != nil {
			return err
		}
		kvs = append(kvs, preInstallValues...)
	}
	// check for global image pull secret
	kvs, err = addGlobalImagePullSecretHelmOverride(log, client, resolvedNamespace, kvs, h.imagePullSecretKeyname)
	if err != nil {
		return err
	}

	// vz-specific chart overrides file
	overridesString, err := h.buildOverridesString(log, client, resolvedNamespace, kvs...)
	if err != nil {
		return err
	}

	// Perform a helm upgrade --install
	_, _, err = upgradeFunc(log, h.releaseName, resolvedNamespace, h.chartDir, h.waitForInstall, dryRun, overridesString, h.valuesFile)
	return err
}

// Upgrade is done by using the helm chart upgrade command.  This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install. Along with the override files in helm_config, we need to generate image overrides using the
// BOM json file.  Each component also has the ability to add additional override parameters.
func (h helmComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	// Resolve the namespace
	namespace := resolveNamespace(h, ns)

	// Check if the component is installed before trying to upgrade
	found, err := helm.IsReleaseInstalled(h.releaseName, namespace)
	if err != nil {
		return err
	}
	if !found {
		log.Infof("Skipping upgrade of component %s since it is not installed", h.releaseName)
		return nil
	}

	// Do the preUpgrade if the function is defined
	if h.preUpgradeFunc != nil && UpgradePrehooksEnabled {
		log.Infof("Running preUpgrade function for %s", h.releaseName)
		err := h.preUpgradeFunc(log, client, h.releaseName, namespace, h.chartDir)
		if err != nil {
			return err
		}
	}

	overridesString, err := h.buildOverridesString(log, client, namespace)
	if err != nil {
		return err
	}

	stdout, err := helm.GetValues(log, h.releaseName, namespace)
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
	_, _, err = upgradeFunc(log, h.releaseName, namespace, h.chartDir, true, dryRun, overridesString, h.valuesFile, tmpFile.Name())
	return err
}

func (h helmComponent) buildOverridesString(log *zap.SugaredLogger, _ clipkg.Client, namespace string, additionalValues ...keyValue) (string, error) {
	// Optionally create a second override file.  This will contain both image overridesString and any additional
	// overridesString required by a component.
	// Get image overridesString unless opt out
	var kvs []keyValue
	var err error
	if !h.ignoreImageOverrides {
		kvs, err = getImageOverrides(h.releaseName)
		if err != nil {
			return "", err
		}
	}

	// Append any additional overridesString for the component (see Keycloak.go for example)
	if h.appendOverridesFunc != nil {
		overrideValues, err := h.appendOverridesFunc(log, h.releaseName, namespace, h.chartDir, []keyValue{})
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
			bldr.WriteString(fmt.Sprintf("%s=%s", kv.key, kv.value))
		}
		overridesString = bldr.String()
	}
	return overridesString, nil
}

// resolveNamespace Resolve/normalize the namespace for a Helm-based component
//
// The need for this stems from an issue with the Verrazzano component and the fact
// that component charts underneath VZ component need to have the ns overridden
func resolveNamespace(h helmComponent, ns string) string {
	namespace := ns
	if h.resolveNamespaceFunc != nil {
		namespace = h.resolveNamespaceFunc(namespace)
	}
	if h.ignoreNamespaceOverride {
		namespace = h.chartNamespace
	}
	return namespace
}

// Get the image overrides from the BOM
func getImageOverrides(subcomponentName string) ([]keyValue, error) {
	// Create a Bom and get the key value overrides
	bom, err := NewBom(DefaultBomFilePath())
	if err != nil {
		return nil, err
	}

	numImages := bom.GetSubcomponentImageCount(subcomponentName)
	if numImages == 0 {
		return []keyValue{}, nil
	}

	kvs, err := bom.buildImageOverrides(subcomponentName)
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
