// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"

	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
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
func (v Verrazzano) Upgrade(log *zap.SugaredLogger, _ clipkg.Client, namespace string) error {
	_, _, err := helm.Upgrade(log, vzReleaseName, resolveNamespace(namespace), VzChartDir(), "")
	return err
}

// resolveNamesapce will return the default verrzzano system namespace unless the namespace
// is specified
func resolveNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return vzDefaultNamespace
}

// VzChartDir returns the chart directory of the verrazzano helm chart on the docker image.
// This can be set by developer to run the operator in development outside of kubernetes
func VzChartDir() string {
	return filepath.Join(config.Get().HelmConfigDir, "charts/verrazzano")
}
