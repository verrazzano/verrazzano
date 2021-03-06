// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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

	// preUpgradeFunc is an optional function to run before upgrading
	preUpgradeFunc preUpgradeFuncSig

	// appendOverridesFunc is an optional function get additional override values
	appendOverridesFunc appendOverridesSig

	// resolveNamespaceFunc is an optional function to process the namespace name
	resolveNamespaceFunc resolveNamespaceSig
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

// preUpgradeFuncSig is the signature for the optional preUgrade function
type preUpgradeFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error

// appendOverridesSig is an optional function called to generate additional overrides.
type appendOverridesSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, kvs []keyValue) ([]keyValue, error)

// resolveNamespaceSig is an optional function called for special namespace processing
type resolveNamespaceSig func(ns string) string

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overrideFile string, overrides string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = helm.Upgrade

// UpgradePrehooksEnabled is needed so that higher level units tests can disable as needed
var UpgradePrehooksEnabled = true

// Name returns the component name
func (h helmComponent) Name() string {
	return h.releaseName
}

// Upgrade is done by using the helm chart upgrade command.  This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install. Along with the override files in helm_config, we need to generate image overrides using the
// BOM json file.  Each component also has the ability to add additional override parameters.
func (h helmComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string) error {
	// Resolve the namespace
	namespace := ns
	if h.resolveNamespaceFunc != nil {
		namespace = h.resolveNamespaceFunc(namespace)
	}
	if h.ignoreNamespaceOverride {
		namespace = h.chartNamespace
	}

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
		err := h.preUpgradeFunc(log, client, h.releaseName, namespace, h.chartDir)
		if err != nil {
			return err
		}
	}

	// Optionally create a second override file.  This will contain both image overrides and any additional
	// overrides required by a component.
	// Get image overrides unless opt out
	var kvs []keyValue
	if !h.ignoreImageOverrides {
		kvs, err = getImageOverrides(h.releaseName)
		if err != nil {
			return err
		}
	}
	// Append any additional overrides for the component (see Keycloak.go for example)
	if h.appendOverridesFunc != nil {
		kvs, err = h.appendOverridesFunc(log, h.releaseName, namespace, h.chartDir, kvs)
		if err != nil {
			return err
		}
	}

	// If there are overrides the create a comma separated string
	var overrides string
	if len(kvs) > 0 {
		bldr := strings.Builder{}
		for i, kv := range kvs {
			if i > 0 {
				bldr.WriteString(",")
			}
			bldr.WriteString(fmt.Sprintf("%s=%s", kv.key, kv.value))
		}
		overrides = bldr.String()
	}

	// Do the upgrade
	_, _, err = upgradeFunc(log, h.releaseName, namespace, h.chartDir, h.valuesFile, overrides)
	return err
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
