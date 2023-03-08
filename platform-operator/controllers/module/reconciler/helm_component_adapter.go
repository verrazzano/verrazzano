// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package reconciler

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	modules2 "github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"k8s.io/apimachinery/pkg/runtime"
)

type helmComponentAdapter struct {
	helmcomp.HelmComponent
	module *modulesv1alpha1.Module
}

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(log vzlog.VerrazzanoLogger, repoURL string, releaseName string, namespace string, chartDirOrName string, chartVersion string, wait bool, dryRun bool, overrides []helm.HelmOverrides) (stdout []byte, stderr []byte, err error)

var (
	_ spi.Component = helmComponentAdapter{}

	upgradeFunc upgradeFuncSig = helm.UpgradeRelease
)

func SetUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func SetDefaultUpgradeFunc() {
	upgradeFunc = helm.UpgradeRelease
}

func NewHelmAdapter(module *modulesv1alpha1.Module) modules2.DelegateReconciler {
	installer := module.Spec.Installer
	chartURL := fmt.Sprintf("%s/%s", installer.HelmChart.Repository.URI, installer.HelmChart.Repository.Path)
	hc := helmcomp.HelmComponent{
		ReleaseName:             module.Name,
		ChartDir:                installer.HelmChart.Name,
		ChartNamespace:          module.ChartNamespace(),
		RepositoryURL:           chartURL,
		ChartVersion:            installer.HelmChart.Version,
		IgnoreNamespaceOverride: true,

		GetInstallOverridesFunc: func(object runtime.Object) interface{} {
			return installer.HelmChart.InstallOverrides.ValueOverrides
		},

		ImagePullSecretKeyname: constants.GlobalImagePullSecName,

		//Dependencies:           nil,
		//PreUpgradeFunc:            nil,
		//AppendOverridesFunc:       nil,
		//ResolveNamespaceFunc:      nil,
		//SupportsOperatorInstall:   false,
		//SupportsOperatorUninstall: false,
		//WaitForInstall:            false,
		//SkipUpgrade:               false,
		//MinVerrazzanoVersion:      "",
		//IngressNames:              nil,
		//Certificates:              nil,
		//AvailabilityObjects:       nil,
	}
	component := helmComponentAdapter{
		HelmComponent: hc,
		module:        module,
	}
	return &Reconciler{
		Component: &component,
	}
}

// Install installs the component using Helm
func (h helmComponentAdapter) Install(context spi.ComponentContext) error {

	var kvs []bom.KeyValue
	// check for global image pull secret
	kvs, err := secret.AddGlobalImagePullSecretHelmOverride(context.Log(), context.Client(), h.ChartNamespace, kvs, h.ImagePullSecretKeyname)
	if err != nil {
		return err
	}

	// vz-specific chart overrides file
	//overrides, err := h.buildCustomHelmOverrides(context, h.ChartNamespace, kvs...)
	//defer vzos.RemoveTempFiles(context.Log().GetZapLogger(), `helm-overrides.*\.yaml`)
	//if err != nil {
	//	return err
	//}

	// Perform an install using the helm upgrade --install command
	_, _, err = upgradeFunc(context.Log(), h.RepositoryURL, h.ReleaseName, h.ChartNamespace, h.ChartDir, h.ChartVersion, h.WaitForInstall, context.IsDryRun(), []helm.HelmOverrides{})
	return err
}

// IsReady Indicates whether a component is available and ready
func (h helmComponentAdapter) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	//releaseAppVersion, err := helm.GetReleaseAppVersion(h.ReleaseName, h.ChartNamespace)
	//if err != nil {
	//	return false
	//}
	//if h.ChartVersion != releaseAppVersion {
	//	return false
	//}

	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, h.ChartNamespace); deployed {
		return true
	}
	return false
}

// TODO: provide override here when we add Enabled hooks to v1beta2 Module
//// IsEnabled Indicates whether a component is enabled for installation
//func (h helmComponentAdapter) IsEnabled(effectiveCR runtime.Object) bool {
//	return true
//}
