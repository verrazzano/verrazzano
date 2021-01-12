// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
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
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

// upgradeFuncSig is needed for unit test override
type upgradeFuncSig func(releaseName string, namespace string, chartDir string, overwriteYaml string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = helm.Upgrade

// Name returns the component name
func (h helmComponent) Name() string {
	return h.releaseName
}

// Upgrade is done by using the helm chart upgrade command.   This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install.
func (h helmComponent) Upgrade(ns string) error {
	namespace := ns
	if h.ignoreNamespaceOverride {
		namespace = h.chartNamespace
	}
	_, _, err := upgradeFunc(h.releaseName, namespace, h.chartDir, h.valuesFile)
	return err
}

func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = helm.Upgrade
}
