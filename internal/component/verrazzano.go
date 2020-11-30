// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	installv1alpha1 "github.com/verrazzano/verrazzano-platform-operator/api/v1alpha1"
	"github.com/verrazzano/verrazzano-platform-operator/internal/helm"
	vz_os "github.com/verrazzano/verrazzano-platform-operator/internal/util/os"
	"path/filepath"
)

const vzReleaseName = "verrazzano"
const vzDefaultNamespace = "verrazzano-system"
const charDir = "install/chart"

// Verrazzano struct
type Verrazzano struct {
}

// Verify that Verrazzano implements Component
var _ Component = Verrazzano{}

// Name returns the component name
func (v Verrazzano) Name() string {
	return "verrazzano"
}

// Upgrade upgrades the component
func (v Verrazzano) Upgrade(cr *installv1alpha1.Verrazzano) error {
	absChartDir := filepath.Join(vz_os.VzRootDir(),charDir)
	err := helm.Upgrade(vzReleaseName, resolveNamespace(cr.Namespace), absChartDir)
	return err
}

// resolveNamesapce will return the default verrzzano system namespace unless the namespace
// is explicity specified
func resolveNamespace(ns string) string {
	if len(ns) > 0  && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}
