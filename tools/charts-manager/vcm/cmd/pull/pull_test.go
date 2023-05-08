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
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/test"
)

const (
	dummyError   = "dummy error"
	flagNotFound = "%s flag not supported by command"
)

func TestNewCmdPatch(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	cmd := NewCmdPull(rc, vcmtesthelpers.FakeHelmChartFileSystem{}, vcmtesthelpers.FakeHelmConfig{})
	assert.NotNil(t, cmd, "command is nil")
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagChartName), fmt.Sprintf(flagNotFound, constants.FlagChartName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagVersionName), fmt.Sprintf(flagNotFound, constants.FlagVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagRepoName), fmt.Sprintf(flagNotFound, constants.FlagRepoName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagDirName), fmt.Sprintf(flagNotFound, constants.FlagDirName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagTargetVersionName), fmt.Sprintf(flagNotFound, constants.FlagTargetVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagUpstreamProvenanceName), fmt.Sprintf(flagNotFound, constants.FlagUpstreamProvenanceName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagPatchName), fmt.Sprintf(flagNotFound, constants.FlagPatchName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagPatchVersionName), fmt.Sprintf(flagNotFound, constants.FlagPatchVersionName))
	assert.Equal(t, buildExample(), cmd.Example)
}

func TestExecPatchCmd(t *testing.T) {
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
		hfs        vcmtesthelpers.FakeHelmChartFileSystem
		helmConfig vcmtesthelpers.FakeHelmConfig
		wantError  error
	}{
		{
			name:      "testChartArgumentNilThrowsError",
			args:      args{chart: "", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagChartName, constants.FlagChartName, constants.FlagChartShorthand),
		},
		{
			name:      "testChartArgumentEmptyThrowsError",
			args:      args{chart: "\n", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagChartName),
		},
		{
			name:      "testVersionArgumentNilThrowsError",
			args:      args{chart: "chart", version: "", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagVersionName, constants.FlagVersionName, constants.FlagVersionShorthand),
		},
		{
			name:      "testVersionArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "\t", chartsDir: "/tmp/charts", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagVersionName),
		},
		{
			name:      "testRepoArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: ""},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagRepoName, constants.FlagRepoName, constants.FlagRepoShorthand),
		},
		{
			name:      "testRepoArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "\n\t"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagRepoName),
		},
		{
			name:      "testDirArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagDirName, constants.FlagDirName, constants.FlagDirShorthand),
		},
		{
			name:      "testDirArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "\n", repo: "https://test"},
			hfs:       vcmtesthelpers.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagDirName),
		},
		{
			name: "testAddAndUpdateChartRepoThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: vcmtesthelpers.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", fmt.Errorf(dummyError)
				},
			},
			wantError: fmt.Errorf(dummyError),
		},
		{
			name: "testDownloadChartThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: vcmtesthelpers.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return fmt.Errorf(dummyError)
				},
			},
			wantError: fmt.Errorf(dummyError),
		},
		{
			name: "testRearrangeChartDirectoryThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: vcmtesthelpers.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
			},
			hfs: vcmtesthelpers.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return fmt.Errorf(dummyError)
				},
			},
			wantError: fmt.Errorf(dummyError),
		},
		{
			name: "testNoSaveUpstreamNoDiffThrowsNoError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", repo: "https://test"},
			helmConfig: vcmtesthelpers.FakeHelmConfig{
				FakeAddAndUpdateChartRepo: func(chart string, repoUrl string) (string, error) {
					return "", nil
				},
				FakeDownloadChart: func(chart string, repo string, version string, targetVersion string, chartDir string) error {
					return nil
				},
			},
			hfs: vcmtesthelpers.FakeHelmChartFileSystem{
				FakeRearrangeChartDirectory: func(chartsDir string, chart string, targetVersion string) error {
					return nil
				},
			},
			wantError: nil,
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
