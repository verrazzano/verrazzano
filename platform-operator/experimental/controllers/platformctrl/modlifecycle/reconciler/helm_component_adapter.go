// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package reconciler

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/common"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/modlifecycle/delegates"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type helmComponentAdapter struct {
	helmcomp.HelmComponent

	// ChartVersion is the version of the helm chart
	ChartVersion string

	// RepositoryURL The name or URL of the repository, e.g., http://myrepo/vz/stable
	RepositoryURL string

	module *modulesv1beta2.ModuleLifecycle
}

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(log vzlog.VerrazzanoLogger, releaseOpts *helm.HelmReleaseOpts, wait bool, dryRun bool) (*release.Release, error)

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

func newHelmAdapter(module *modulesv1beta2.ModuleLifecycle, sw client.StatusWriter) delegates.DelegateLifecycleReconciler {
	installer := module.Spec.Installer
	chartInfo := installer.HelmRelease.ChartInfo
	chartURL := fmt.Sprintf("%s/%s", installer.HelmRelease.Repository.URI, chartInfo.Path)
	hc := helmcomp.HelmComponent{
		ReleaseName:             module.Name,
		ChartDir:                chartInfo.Path,
		ChartNamespace:          module.ChartNamespace(),
		IgnoreNamespaceOverride: true,

		//GetInstallOverridesFunc: func(object runtime.Object) interface{} {
		//	return v1beta1.InstallOverrides{
		//		ValueOverrides: copyOverrides(installer.HelmRelease.Overrides),
		//	}
		//},

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
		RepositoryURL: chartURL,
		ChartVersion:  chartInfo.Version,
		module:        module,
	}
	return &helmDelegateReconciler{
		StatusWriter: sw,
		comp:         &component,
	}
}

// Install installs the component using Helm
func (h helmComponentAdapter) Install(context spi.ComponentContext) error {
	// Perform a Helm install using the helm upgrade --install command
	helmOverrides, err := common.BuildHelmOverrides(context.Log(), context.Client(), h.ReleaseName, h.ChartNamespace, h.module.Spec.Installer.HelmRelease.Overrides)
	if err != nil {
		return err
	}
	var opts = &helm.HelmReleaseOpts{
		RepoURL:      h.RepositoryURL,
		ReleaseName:  h.ReleaseName,
		Namespace:    h.ChartNamespace,
		ChartPath:    h.ChartDir,
		ChartVersion: h.ChartVersion,
		Overrides:    helmOverrides,
		//Username:     "",
		//Password:     "",
	}
	_, err = upgradeFunc(context.Log(), opts, h.WaitForInstall, context.IsDryRun())
	return err
}

func (h helmComponentAdapter) Upgrade(context spi.ComponentContext) error {
	// TODO: examine HelmComponent.Upgrade() to see what kind of hooks are missing/required
	return h.Install(context)
}

// IsReady Indicates whether a component is available and ready
func (h helmComponentAdapter) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	// TODO: see if we need any of this nonsense below
	releaseAppVersion, err := helm.GetReleaseAppVersion(h.ReleaseName, h.ChartNamespace)
	if err != nil {
		return false
	}
	if h.ChartVersion != releaseAppVersion {
		return false
	}

	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, h.ChartNamespace); deployed {
		return true
	}
	return false
}

// IsEnabled ModuleLifecycle objects are always enabled; if a Module is disabled the ModuleLifecycle resource doesn't exist
func (h helmComponentAdapter) IsEnabled(_ runtime.Object) bool {
	return true
}
