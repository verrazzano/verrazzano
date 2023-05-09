// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fakes

import (
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// FakeHelmChartFileSystem is a fake implemntation of fs.ChartFileSystem.
type FakeHelmChartFileSystem struct {
	FakeRearrangeChartDirectory    func(string, string, string) error
	FakeSaveUpstreamChart          func(string, string, string, string) error
	FakeSaveChartProvenance        func(string, *helm.ChartProvenance, string, string) error
	FakeGeneratePatchFile          func(string, string, string) (string, error)
	FakeGeneratePatchWithSourceDir func(string, string, string, string) (string, error)
	FakeFindChartVersionToPatch    func(string, string, string) (string, error)
	FakeApplyPatchFile             func(string, helpers.VZHelper, string, string, string) (bool, error)
}

// FakeHelmConfig is a fake implementation of helm.HelmConfig.
type FakeHelmConfig struct {
	FakeAddAndUpdateChartRepo func(string, string) (string, error)
	FakeDownloadChart         func(string, string, string, string, string) error
	FakeGetChartProvenance    func(string, string, string) (*helm.ChartProvenance, error)
}

// RearrangeChartDirectory is a fake implementation of fs.ChartFileSystem.RearrangeChartDirectory.
func (hfs FakeHelmChartFileSystem) RearrangeChartDirectory(chartsDir string, chart string, targetVersion string) error {
	return hfs.FakeRearrangeChartDirectory(chartsDir, chart, targetVersion)
}

// SaveUpstreamChart is a fake implementation of fs.ChartFileSystem.SaveUpstreamChart.
func (hfs FakeHelmChartFileSystem) SaveUpstreamChart(chartsDir string, chart string, version string, targetVersion string) error {
	return hfs.FakeSaveUpstreamChart(chartsDir, chart, version, targetVersion)
}

// SaveChartProvenance is a fake implementation of fs.ChartFileSystem.SaveChartProvenance.
func (hfs FakeHelmChartFileSystem) SaveChartProvenance(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
	return hfs.FakeSaveChartProvenance(chartsDir, chartProvenance, chart, targetVersion)
}

// GeneratePatchFile is a fake implementation of fs.ChartFileSystem.GeneratePatchFile.
func (hfs FakeHelmChartFileSystem) GeneratePatchFile(chartsDir string, chart string, version string) (string, error) {
	return hfs.FakeGeneratePatchFile(chartsDir, chart, version)
}

// GeneratePatchWithSourceDir is a fake implementation of fs.ChartFileSystem.GeneratePatchWithSourceDir.
func (hfs FakeHelmChartFileSystem) GeneratePatchWithSourceDir(chartsDir string, chart string, version string, sourceDir string) (string, error) {
	return hfs.FakeGeneratePatchWithSourceDir(chartsDir, chart, version, sourceDir)
}

// FindChartVersionToPatch is a fake implementation of fs.ChartFileSystem.FindChartVersionToPatch.
func (hfs FakeHelmChartFileSystem) FindChartVersionToPatch(chartsDir string, chart string, version string) (string, error) {
	return hfs.FakeFindChartVersionToPatch(chartsDir, chart, version)
}

// ApplyPatchFile is a fake implementation of fs.ChartFileSystem.ApplyPatchFile.
func (hfs FakeHelmChartFileSystem) ApplyPatchFile(chartsDir string, vzHelper helpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
	return hfs.FakeApplyPatchFile(chartsDir, vzHelper, chart, version, patchFile)
}

// AddAndUpdateChartRepo is a fake implementation of helm.HelmConfig.AddAndUpdateChartRepo.
func (h FakeHelmConfig) AddAndUpdateChartRepo(chart string, repoURL string) (string, error) {
	return h.FakeAddAndUpdateChartRepo(chart, repoURL)
}

// DownloadChart is a fake implementation of helm.HelmConfig.DownloadChart.
func (h FakeHelmConfig) DownloadChart(chart string, repo string, version string, targetVersion string, chartDir string) error {
	return h.FakeDownloadChart(chart, repo, version, targetVersion, chartDir)
}

// GetChartProvenance is a fake implementation of helm.HelmConfig.GetChartProvenance.
func (h FakeHelmConfig) GetChartProvenance(chart string, repo string, version string) (*helm.ChartProvenance, error) {
	return h.FakeGetChartProvenance(chart, repo, version)
}
