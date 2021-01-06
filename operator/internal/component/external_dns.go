// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
config2 "github.com/verrazzano/verrazzano/operator/internal/config"
"github.com/verrazzano/verrazzano/operator/internal/util/helm"
"path/filepath"
)

const vzReleaseName = "verrazzano"
const vzDefaultNamespace = "verrazzano-system"

// Verrazzano struct needed to implement interface
type Verrazzano struct {
}

// Verify that Verrazzano implements Component
var _ Component = Verrazzano{}

// Name returns the component name
func (v Verrazzano) Name() string {
	return "verrazzano"
}

// Upgrade external dns
func (v Verrazzano) Upgrade(namespace string) error {
	_, _, err := helm.Upgrade(vzReleaseName, resolveNamespace(namespace), VzChartDir())
	return err
}

// resolveNamesapce will return the default verrazzano system namespace unless the namespace
// is explicity specified
func resolveNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}

// VzChartDir returns the chart directory of the verrazzano helm chart on the docker image.
// This can be set by developer to run the operator in development outside of kubernetes
func VzChartDir() string {
	dir := config2.Get().VerrazzanoInstallDir
	return filepath.Join(dir + "/chart")
}
