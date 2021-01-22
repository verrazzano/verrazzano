// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
)

const nginxReleaseName = "ingress-controller"
const nginxDefaultNamespace = "ingress-nginx"

// Nginx struct needed to implement interface
type Nginx struct {
}

// Verify that Nginx implements Component
var _ Component = Nginx{}

// Name returns the component name
func (v Nginx) Name() string {
	return "ingress-nginx"
}

// Upgrade upgrades the NGINX ingress controller.
// Upgrade is done by using the helm chart upgrade command.   This command will apply the latest chart
// that is included in the operator image, while retaining any helm value overrides that were applied during
// install.
func (v Nginx) Upgrade(_ clipkg.Client, namespace string) error {
	_, _, err := helm.Upgrade(nginxReleaseName, nginxNamespace(namespace), nginxChartDir(), nginxOverrideYamlFile())
	return err
}

// nginxNamespace will return the default NGINX namespace unless the namespace
// is explicitly specified
func nginxNamespace(ns string) string {
	if len(ns) > 0 && ns != "default" {
		return ns
	}
	return nginxDefaultNamespace
}

// nginxChartDir returns the chart directory of the NGINX ingress controller helm chart.
func nginxChartDir() string {
	dir := config.Get().ThirdpartyChartsDir
	return filepath.Join(dir + "/ingress-nginx")
}

// nginxOverrideYamlFile returns the override yaml file to be used with the NGINX ingress controller helm chart.
func nginxOverrideYamlFile() string {
	dir := config.Get().HelmConfigDir
	return filepath.Join(dir + "/overrides/ingress-nginx-values.yaml")
}
