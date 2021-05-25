// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

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

	// valuesFile is the helm chart values override file
	valuesFile string

	// preUpgradeFunc is an optional function to run before upgrading
	preUpgradeFunc preUpgradeFuncSig
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

// preUpgradeFuncSig is the signature for the optional preUgrade function
type preUpgradeFuncSig func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error

// upgradeFuncSig is needed for unit test override
type upgradeFuncSig func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overrideFile []string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = helm.Upgrade

// Name returns the component name
func (h helmComponent) Name() string {
	return h.releaseName
}

// UpgradePrehooksEnabled is needed so that higher level units tests can disable as needed
var UpgradePrehooksEnabled = true

// Upgrade is done by using the helm chart upgrade command.   This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install.
func (h helmComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string) error {
	namespace := ns
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

	// Create and populate the image override file.
	f, err := ioutil.TempFile("", "vz-images")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = writeImageOverrides(h.releaseName, f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	// Do the upgrade, passing in the override files
	overrideFiles := []string {
		h.valuesFile,
		f.Name(),
	}
	_, _, err = upgradeFunc(log, h.releaseName, namespace, h.chartDir, overrideFiles)
	return err
}

// Write the image key:value pairs to the override file
func writeImageOverrides(subcomponentName string, w io.Writer) error {
	// Get the key value pair
	bom, err := NewBom(DefaultBomFilePath())
	if err != nil {
		return err
	}
	kvs, err := bom.buildImageOverrides(subcomponentName)
	if err != nil {
		return err
	}
	// Override entries are in the helm format of key: value
	for _, kv := range kvs {
		io.WriteString(w, fmt.Sprintf("%s: %s\n",kv.key, kv.value))
	}
	return nil
}


func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = helm.Upgrade
}
