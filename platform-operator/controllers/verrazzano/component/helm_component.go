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

	overrideFiles := []string{}
	if h.valuesFile != "" {
		overrideFiles = append(overrideFiles, h.valuesFile)
	}

	// If there are image overrides the get them and write to an override file
	kvs, err := getImageOverrides(h.releaseName)
	if err != nil {
		return err
	}
	if len(kvs) > 0 {
		// Create and populate the image override file.
		f, err := ioutil.TempFile("", "vz-images")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())

		// Write the override entries then close the file
		for _, kv := range kvs {
			io.WriteString(f, fmt.Sprintf("%s: %s\n", kv.key, kv.value))
		}
		err = f.Close()
		if err != nil {
			return err
		}
		overrideFiles = append(overrideFiles, f.Name())
	}

	// Do the upgrade, passing in the override files
	_, _, err = upgradeFunc(log, h.releaseName, namespace, h.chartDir, overrideFiles)
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
