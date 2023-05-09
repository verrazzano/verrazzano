// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diff

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/fakes"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
)

// TestNewCmdDiff tests that function NewCmdDiff creates diff cmd with correct flags
// GIVEN a call to NewCmdDiff
//
//	WHEN correct arguments are passed
//	THEN the diff cmd instance created contains all the required flags.
func TestNewCmdDiff(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	cmd := NewCmdDiff(rc, fakes.FakeHelmChartFileSystem{})
	assert.NotNil(t, cmd, "command is nil")
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagChartName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagChartName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagVersionName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagDirName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagDirName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagDiffSourceName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagDiffSourceName))
	assert.Equal(t, buildExample(), cmd.Example)
}

// TestExecDiffCmd tests the execution of diff command
// GIVEN a call to NewCmdDiff and then executing the resulting diff command with specific parameters to generate
// a diff of a chart directory against a source directory
//
//	WHEN invalid arguments are passed
//	THEN the cmd execution results in an error.
//
//	WHEN the patch generation returns an error
//	THEN the cmd execution results in an error.
//
//	WHEN the patch generation does not return an error or no patch is generated
//	THEN the cmd execution does not result in an error.
func TestExecDiffCmd(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	type args struct {
		chart     string
		version   string
		chartsDir string
		sourceDir string
	}
	tests := []struct {
		name      string
		args      args
		hfs       fakes.FakeHelmChartFileSystem
		wantError error
	}{
		{
			name:      "testChartArgumentNilThrowsError",
			args:      args{chart: "", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagChartName, constants.FlagChartName, constants.FlagChartShorthand),
		},
		{
			name:      "testChartArgumentEmptyThrowsError",
			args:      args{chart: "\n", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagChartName),
		},
		{
			name:      "testVersionArgumentNilThrowsError",
			args:      args{chart: "chart", version: "", chartsDir: "/tmp/charts", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagVersionName, constants.FlagVersionName, constants.FlagVersionShorthand),
		},
		{
			name:      "testVersionArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "\t", chartsDir: "/tmp/charts", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagVersionName),
		},
		{
			name:      "testDirArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagDirName, constants.FlagDirName, constants.FlagDirShorthand),
		},
		{
			name:      "testDirArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "\n", sourceDir: "/tmp/preveious"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagDirName),
		},
		{
			name:      "testSourceArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: ""},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagDiffSourceName, constants.FlagDiffSourceName, constants.FlagDiffSourceShorthand),
		},
		{
			name:      "testSourceArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "\t\n"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagDiffSourceName),
		},
		{
			name: "testGeneratePatchThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "/tmp/previous"},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeGeneratePatchWithSourceDir: func(chartsDir string, chart string, version string, sourceDir string) (string, error) {
					return "", fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testGeneratePatchRetursNoPatchFile",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "/tmp/previous"},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeGeneratePatchWithSourceDir: func(chartsDir string, chart string, version string, sourceDir string) (string, error) {
					return "", nil
				},
			},
			wantError: nil,
		},
		{
			name: "testGeneratePatchRetursValidPatchFile",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", sourceDir: "/tmp/previous"},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeGeneratePatchWithSourceDir: func(chartsDir string, chart string, version string, sourceDir string) (string, error) {
					return "/tmp/patchfile", nil
				},
			},
			wantError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdDiff(rc, tt.hfs)
			cmd.PersistentFlags().Set(constants.FlagChartName, tt.args.chart)
			cmd.PersistentFlags().Set(constants.FlagVersionName, tt.args.version)
			cmd.PersistentFlags().Set(constants.FlagDirName, tt.args.chartsDir)
			cmd.PersistentFlags().Set(constants.FlagDiffSourceName, tt.args.sourceDir)
			err := cmd.Execute()
			if err != nil && tt.wantError == nil {
				t.Errorf("diff exec with args %v resulted in error %v", tt.args, err)
			}

			if err != nil && tt.wantError != nil && err.Error() != tt.wantError.Error() {
				t.Errorf("diff exec with args %v resulted in error %v, expected %v", tt.args, err, tt.wantError)
			}

			if err == nil && tt.wantError != nil {
				t.Errorf("diff exec with args %v resulted in no error, expected %v", tt.args, tt.wantError)
			}
		})
	}
}
