// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
	vz_os "github.com/verrazzano/verrazzano/operator/internal/util/os"
	"path/filepath"
)

const vzReleaseName = "verrazzano"
const vzDefaultNamespace = "verrazzano-system"
const charDir = "operator/scripts/install/chart"

// Verrazzano struct needed to implement interface
type Verrazzano struct {
}

// Verify that Verrazzano implements Component
var _ Component = Verrazzano{}

// Name returns the component name
func (v Verrazzano) Name() string {
	return "verrazzano"
}

// Upgrade upgrades all of the Verrazzano home-grown components including the following:
//  Verrazzano operator
//  Verrazzano WLS micro-operator
//  Verrazzano COH micro-operator
//  Verrazzano Helidon micro-operator
//  Verrazzano Cluster micro-operator
//  Verrazzano VMO operator
//  Verrazzano admission controller
//
// Upgrade is done by using the helm chart upgrade command.   This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install.
func (v Verrazzano) Upgrade(namespace string) error {
	absChartDir := filepath.Join(vz_os.VzRootDir(), charDir)
	_, _, err := helm.Upgrade(vzReleaseName, resolveNamespace(namespace), absChartDir)
	return err
}

// resolveNamesapce will return the default verrzzano system namespace unless the namespace
// is explicity specified
func resolveNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}
