// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pull

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/fakes"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
	vzhelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// TestNewCmdPull tests that function NewCmdPull creates pull cmd with correct flags
// GIVEN a call to NewCmdPull
//
//	WHEN correct arguments are passed
//	THEN the pull cmd instance created contains all the required flags.
func TestNewCmdPull(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	cmd := NewCmdPull(rc, fakes.FakeHelmChartFileSystem{}, fakes.FakeHelmConfig{})
	assert.NotNil(t, cmd, "command is nil")
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagChartName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagChartName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagVersionName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagRepoName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagRepoName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagDirName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagDirName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagTargetVersionName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagTargetVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagUpstreamProvenanceName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagUpstreamProvenanceName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagPatchName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagPatchName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagPatchVersionName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagPatchVersionName))
	assert.Equal(t, buildExample(), cmd.Example)
}

// TestExecPullCmd tests the execution of pull command
// GIVEN a call to NewCmdPull and then executing the resulting pull command with specific parameters to pull a chart
//
//		WHEN invalid arguments are passed
//		THEN the cmd execution results in an error.
//
//		WHEN adding/updating helm repo results in an error
//		THEN the cmd execution results in an error.
//
//		WHEN downloading helm chart results in an error
//		THEN the cmd execution results in an error.
//
//		WHEN rearranging the chart directory results in an error
//		THEN the cmd execution results in an error.
//
//		WHEN the chart is downloaded to correct directory and no upstream was to be saved
//	 	AND no patching from a previous version is to be applied
//		THEN the cmd execution does not result in an error.
//
//		WHEN saving upstream chart returns in error
//		THEN the cmd execution results in an error.
//
//		WHEN saving upstream chart returns in error
//		THEN the cmd execution results in an error.
//
//		WHEN creating the chart provenance data returns in error
//		THEN the cmd execution results in an error.
//
//		WHEN saving the chart provenance file returns in error
//		THEN the cmd execution results in an error.
//
//		WHEN saving upstream chart is successful and chart provenance is saved and patch flag is false
//		THEN the cmd execution results in no error.
//
//		WHEN patch flag is true and finding the version to generate the patch from is unsuccessful
//		THEN the cmd execution results in an error.
//
//		WHEN patch flag is true and generting the patch is unsuccessful
//		THEN the cmd execution results in an error.
//
//		WHEN patch flag is true and no patch file is generated because there are no diffs
//		THEN the cmd execution results in no error.
//
//		WHEN patch flag is true and patch file is generated but applying the patch is unsuccessful
//		THEN the cmd execution results in error.
//
//		WHEN patch flag is true and a rejects file is generated while applying the patch
//		THEN the cmd execution results in error.
func TestExecPullCmd(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	type args struct {
		chart         string
		version       string
		chartsDir     string
		repo          string
		targetVersion string
		upstreamProv  bool
		patch         bool
		patchVersion  string
	}
	tests := []struct {
		name       string
		args       args
		hfs        fakes.FakeHelmChartFileSystem
		helmConfig fakes.FakeHelmConfig
		wantError  error
	}{
		{
			name:      "testChartArgumentNilThrowsError",
			args:      args{chart: "", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagChartName, constants.FlagChartName, constants.FlagChartShorthand),
		},
		{
			name:      "testChartArgumentEmptyThrowsError",
			args:      args{chart: "\n", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagChartName),
		},
		{
			name:      "testVersionArgumentNilThrowsError",
			args:      args{chart: "chart", version: "", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagVersionName, constants.FlagVersionName, constants.FlagVersionShorthand),
		},
		{
			name:      "testVersionArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "\t", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagVersionName),
		},
		{
			name:      "testRepoArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: ""},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagRepoName, constants.FlagRepoName, constants.FlagRepoShorthand),
		},
		{
			name:      "testRepoArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "\n\t"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagRepoName),
		},
		{
			name:      "testDirArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagDirName, constants.FlagDirName, constants.FlagDirShorthand),
		},
		{
			name:      "testDirArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "\n", repo: "https://test"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagDirName),
		},
		{
			name: "testAddAndUpdateChartRepoThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testDownloadChartThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testRearrangeChartDirectoryThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testNoSaveUpstreamNoDiffThrowsNoError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
			},
			wantError: nil,
		},
		{
			name: "testSaveUpstreamThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testGetChartProvenanceThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testSaveChartProvenanceThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testNoPatchDiffThrowsNoError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
			},
			wantError: nil,
		},
		{
			name: "testFindChartVersionToPatchThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true, patch: true},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
				FakeFindChartVersionToPatch: func(chartsDir string, chart string, version string) (string, error) {
					return "", fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(ErrPatchVersionNotFound, fmt.Errorf(vcmtesthelpers.DummyError)),
		},
		{
			name: "testGeneratePatchFileThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true, patch: true, patchVersion: "x.y.z"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
				FakeGeneratePatchFile: func(chartsDir string, chart string, version string) (string, error) {
					return "", fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(ErrPatchNotGenerated, fmt.Errorf(vcmtesthelpers.DummyError)),
		},
		{
			name: "testNoPatchFileGeneratedNoError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true, patch: true, patchVersion: "x.y.z"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
				FakeGeneratePatchFile: func(chartsDir string, chart string, version string) (string, error) {
					return "", nil
				},
			},
			wantError: nil,
		},
		{
			name: "testApplyPatchFileThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true, patch: true, patchVersion: "x.y.z"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
				FakeGeneratePatchFile: func(chartsDir string, chart string, version string) (string, error) {
					return "dummyfile", nil
				},
				FakeApplyPatchFile: func(chartsDir string, vzHelper vzhelpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
					return false, fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(ErrPatchNotApplied, "dummyfile", fmt.Errorf(vcmtesthelpers.DummyError)),
		},
		{
			name: "testRejectsFileGeneratedThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test", upstreamProv: true, patch: true, patchVersion: "x.y.z"},
			helmConfig: fakes.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
				FakeGetChartProvenance: func(chart string, repo string, version string) (*helm.ChartProvenance, error) {
					return nil, nil
				},
			},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
				FakeSaveUpstreamChart: func(chartsDir string, chart string, version string, targetVersion string) error {
					return nil
				},
				FakeSaveChartProvenance: func(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
					return nil
				},
				FakeGeneratePatchFile: func(chartsDir string, chart string, version string) (string, error) {
					return "dummyfile", nil
				},
				FakeApplyPatchFile: func(chartsDir string, vzHelper vzhelpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
					return true, nil
				},
			},
			wantError: fmt.Errorf(ErrPatchReview, "dummyfile"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdPull(rc, tt.hfs, tt.helmConfig)
			cmd.PersistentFlags().Set(constants.FlagChartName, tt.args.chart)
			cmd.PersistentFlags().Set(constants.FlagVersionName, tt.args.version)
			cmd.PersistentFlags().Set(constants.FlagDirName, tt.args.chartsDir)
			cmd.PersistentFlags().Set(constants.FlagRepoName, tt.args.repo)
			cmd.PersistentFlags().Set(constants.FlagPatchFileName, tt.args.targetVersion)
			cmd.PersistentFlags().Lookup(constants.FlagUpstreamProvenanceName).Value.Set(strconv.FormatBool(tt.args.upstreamProv))
			cmd.PersistentFlags().Lookup(constants.FlagPatchName).Value.Set(strconv.FormatBool(tt.args.patch))
			cmd.PersistentFlags().Set(constants.FlagPatchVersionName, tt.args.patchVersion)
			err := cmd.Execute()
			if err != nil && tt.wantError == nil {
				t.Errorf("pull exec with args %v resulted in error %v", tt.args, err)
			}

			if err != nil && tt.wantError != nil && err.Error() != tt.wantError.Error() {
				t.Errorf("pull exec with args %v resulted in error %v, expected %v", tt.args, err, tt.wantError)
			}

			if err == nil && tt.wantError != nil {
				t.Errorf("pull exec with args %v resulted in no error, expected %v", tt.args, tt.wantError)
			}
		})
	}
}
