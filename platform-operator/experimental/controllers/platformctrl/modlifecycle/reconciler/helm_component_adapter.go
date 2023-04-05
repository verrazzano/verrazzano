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
type upgradeFuncSig func(log vzlog.VerrazzanoLogger, releaseOpts *common.HelmReleaseOpts, wait bool, dryRun bool) (*release.Release, error)

var (
	_ spi.Component = helmComponentAdapter{}

	upgradeFunc upgradeFuncSig = common.UpgradeRelease
)

func SetUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func SetDefaultUpgradeFunc() {
	upgradeFunc = common.UpgradeRelease
}

func newHelmAdapter(mlc *modulesv1beta2.ModuleLifecycle, sw client.StatusWriter) delegates.DelegateLifecycleReconciler {
	installer := mlc.Spec.Installer
	chartInfo := installer.HelmRelease.ChartInfo
	chartURL := fmt.Sprintf("%s/%s", installer.HelmRelease.Repository.URI, chartInfo.Path)
	hc := helmcomp.HelmComponent{
		ReleaseName:             mlc.GetReleaseName(),
		ChartDir:                chartInfo.Path,
		ChartNamespace:          mlc.ChartNamespace(),
		IgnoreNamespaceOverride: true,
		ImagePullSecretKeyname:  constants.GlobalImagePullSecName,
	}
	component := helmComponentAdapter{
		HelmComponent: hc,
		RepositoryURL: chartURL,
		ChartVersion:  chartInfo.Version,
		module:        mlc,
	}
	return &helmDelegateReconciler{
		StatusWriter: sw,
		comp:         &component,
	}
}

// Install installs the component using Helm
func (h helmComponentAdapter) Install(context spi.ComponentContext) error {
	// Perform a Helm install using the helm upgrade --install command
	helmRelease := h.module.Spec.Installer.HelmRelease
	helmOverrides, err := common.ConvertToHelmOverrides(context.Log(), context.Client(), helmRelease.Name, helmRelease.Namespace, helmRelease.Overrides)
	if err != nil {
		return err
	}
	var opts = &common.HelmReleaseOpts{
		RepoURL:      h.RepositoryURL,
		ReleaseName:  h.ReleaseName,
		Namespace:    h.ChartNamespace,
		ChartPath:    helmRelease.ChartInfo.Name,
		ChartVersion: helmRelease.ChartInfo.Version,
		Overrides:    helmOverrides,
		// TBD -- pull from a secret ref?
		//Username:     "",
		//Password:     "",
	}
	_, err = upgradeFunc(context.Log(), opts, h.WaitForInstall, context.IsDryRun())
	return err
}

func (h helmComponentAdapter) Upgrade(context spi.ComponentContext) error {
	return h.Install(context)
}

// IsReady Indicates whether a component is available and ready
func (h helmComponentAdapter) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	deployed, err := helm.IsReleaseDeployed(h.ReleaseName, h.ChartNamespace)
	if err != nil {
		context.Log().ErrorfThrottled("Error occurred checking release deloyment: %v", err.Error())
		return false
	}

	releaseMatches := h.releaseVersionMatches(context.Log())

	// The helm release exists and is at the correct version
	return deployed && releaseMatches
}

func (h helmComponentAdapter) releaseVersionMatches(log vzlog.VerrazzanoLogger) bool {
	releaseChartVersion, err := common.GetReleaseChartVersion(h.ReleaseName, h.ChartNamespace)
	if err != nil {
		log.ErrorfThrottled("Error occurred getting release chart version: %v", err.Error())
		return false
	}
	return h.ChartVersion == releaseChartVersion
}

func (h helmComponentAdapter) Uninstall(context spi.ComponentContext) error {
	releaseName := h.module.GetReleaseName()
	deployed, err := helm.IsReleaseDeployed(releaseName, h.ChartNamespace)
	if err != nil {
		return err
	}
	if !deployed {
		context.Log().Infof("%s already uninstalled", h.Name())
		return nil
	}
	err = helm.Uninstall(context.Log(), releaseName, h.ChartNamespace, context.IsDryRun())
	if err != nil {
		context.Log().Errorf("Error uninstalling %s/%s, error: %s", h.ChartNamespace, h.Name(), err.Error())
		return err
	}
	return nil
}

// IsEnabled ModuleLifecycle objects are always enabled; if a Module is disabled the ModuleLifecycle resource doesn't exist
func (h helmComponentAdapter) IsEnabled(_ runtime.Object) bool {
	return true
}
