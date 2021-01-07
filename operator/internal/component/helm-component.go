// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
)

// Verrazzano struct needed to implement interface
type helmComponent struct {
	// The helm chart release name
	releaseName string

	// The helm chart directory
	chartDir string

	// The namespace passed to the helm command
	chartNamespace string

	// The namespaceHardcoded bool indicates that a component has a hardcoded namespace
	// and ignores the namespace param passed to Upgrade
	namespaceHardcoded bool

	// The helm chart values override file
	valuesFile string
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

// Name returns the component name
func (h helmComponent) Name() string {
	return h.releaseName
}

// Upgrade is done by using the helm chart upgrade command.   This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install.
func (h helmComponent) Upgrade(ns string) error {
	namespace := ns
	if h.namespaceHardcoded {
		namespace = h.chartNamespace
	}
	_, _, err := helm.Upgrade(h.releaseName, namespace, h.chartDir, h.valuesFile)
	return err
}
