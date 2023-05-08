// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package test

import (
	"os"

	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type FakeHelmChartFileSystem struct {
	FakeRearrangeChartDirectory    func(string, string, string) error
	FakeSaveUpstreamChart          func(string, string, string, string) error
	FakeSaveChartProvenance        func(string, *helm.ChartProvenance, string, string) error
	FakeGeneratePatchFile          func(string, string, string) (string, error)
	FakeGeneratePatchWithSourceDir func(string, string, string, string) (string, error)
	FakeFindChartVersionToPatch    func(string, string, string) (string, error)
	FakeApplyPatchFile             func(string, helpers.VZHelper, string, string, string) (bool, error)
}

type FakeHelmConfig struct {
	FakeAddAndUpdateChartRepo func(string, string) (string, error)
	FakeDownloadChart         func(string, string, string, string, string) error
	FakeGetChartProvenance    func(string, string, string) (*helm.ChartProvenance, error)
}

func (hfs FakeHelmChartFileSystem) RearrangeChartDirectory(chartsDir string, chart string, targetVersion string) error {
	return hfs.FakeRearrangeChartDirectory(chartsDir, chart, targetVersion)
}

func (hfs FakeHelmChartFileSystem) SaveUpstreamChart(chartsDir string, chart string, version string, targetVersion string) error {
	return hfs.FakeSaveUpstreamChart(chartsDir, chart, version, targetVersion)
}

func (hfs FakeHelmChartFileSystem) SaveChartProvenance(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
	return hfs.FakeSaveChartProvenance(chartsDir, chartProvenance, chart, targetVersion)
}

func (hfs FakeHelmChartFileSystem) GeneratePatchFile(chartsDir string, chart string, version string) (string, error) {
	return hfs.FakeGeneratePatchFile(chartsDir, chart, version)

}

func (hfs FakeHelmChartFileSystem) GeneratePatchWithSourceDir(chartsDir string, chart string, version string, sourceDir string) (string, error) {
	return hfs.FakeGeneratePatchWithSourceDir(chartsDir, chart, version, sourceDir)
}

func (hfs FakeHelmChartFileSystem) FindChartVersionToPatch(chartsDir string, chart string, version string) (string, error) {
	return hfs.FakeFindChartVersionToPatch(chartsDir, chart, version)
}

func (hfs FakeHelmChartFileSystem) ApplyPatchFile(chartsDir string, vzHelper helpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
	return hfs.FakeApplyPatchFile(chartsDir, vzHelper, chart, version, patchFile)
}

func (h FakeHelmConfig) AddAndUpdateChartRepo(chart string, repoUrl string) (string, error) {
	return h.FakeAddAndUpdateChartRepo(chart, repoUrl)
}

func (h FakeHelmConfig) DownloadChart(chart string, repo string, version string, targetVersion string, chartDir string) error {
	return h.FakeDownloadChart(chart, repo, version, targetVersion, chartDir)
}

func (h FakeHelmConfig) GetChartProvenance(chart string, repo string, version string) (*helm.ChartProvenance, error) {
	return h.FakeGetChartProvenance(chart, repo, version)
}

func ContextSetup() (*testhelpers.FakeRootCmdContext, func(), error) {
	stdoutFile, stderrFile, err := createStdTempFiles()
	if err != nil {
		return nil, nil, err
	}
	return testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile}), func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}, nil
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles() (*os.File, *os.File, error) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	if err != nil {
		return nil, nil, err
	}

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	if err != nil {
		return nil, nil, err
	}

	return stdoutFile, stderrFile, nil
}
