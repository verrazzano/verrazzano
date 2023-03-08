// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package reconciler

import (
	"fmt"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	modules2 "github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

type helmComponentAdapter struct {
	helmcomp.HelmComponent

	// ChartVersion is the version of the helm chart
	ChartVersion string

	// Repository The name or URL of the repository
	Repository string
}

var _ spi.Component = helmComponentAdapter{}

func NewHelmAdapter(module *modulesv1alpha1.Module) modules2.DelegateReconciler {
	installer := module.Spec.Installer
	chartURL := fmt.Sprintf("%s/%s", installer.HelmChart.Repository.URI, installer.HelmChart.Repository.Path,
		installer.HelmChart.Name)
	hc := helmcomp.HelmComponent{
		ReleaseName:    module.Name,
		ChartDir:       installer.HelmChart.Name,
		ChartNamespace: module.ChartNamespace(),
		Repository:     chartURL,
		ChartVersion:   installer.HelmChart.Version,
		//Dependencies:           nil,

		//PreUpgradeFunc:            nil,
		//AppendOverridesFunc:       nil,
		//GetInstallOverridesFunc:   nil,
		//ResolveNamespaceFunc:      nil,
		//SupportsOperatorInstall:   false,
		//SupportsOperatorUninstall: false,
		//WaitForInstall:            false,
		ImagePullSecretKeyname: constants.GlobalImagePullSecName,
		//SkipUpgrade:               false,
		//MinVerrazzanoVersion:      "",
		//IngressNames:              nil,
		//Certificates:              nil,
		//AvailabilityObjects:       nil,
	}
	helmcomp.SetForModule(&hc, module)
	component := helmComponentAdapter{
		HelmComponent: hc,
	}
	return &Reconciler{
		Component: &component,
	}
}
