// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package patch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/fakes"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
	vzhelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// TestNewCmdPatch tests that function NewCmdPatch creates patch cmd with correct flags
// GIVEN a call to NewCmdPatch
//
//	WHEN correct arguments are passed
//	THEN the patch cmd instance created contains all the required flags.
func TestNewCmdPatch(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	cmd := NewCmdPatch(rc, fakes.FakeHelmChartFileSystem{})
	assert.NotNil(t, cmd, "command is nil")
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagChartName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagChartName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagVersionName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagVersionName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagDirName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagDirName))
	assert.NotNil(t, cmd.PersistentFlags().Lookup(constants.FlagPatchFileName), fmt.Sprintf(vcmtesthelpers.FlagNotFound, constants.FlagPatchFileName))
	assert.Equal(t, buildExample(), cmd.Example)
}

// TestExecPatchCmd tests the execution of patch command
// GIVEN a call to NewCmdPatch and then executing the resulting patch command with specific parameters to apply
// a patch on a chart from a given file
//
//	WHEN invalid arguments are passed
//	THEN the cmd execution results in an error.
//
//	WHEN applying the patch results in an error
//	THEN the cmd execution results in an error.
//
//	WHEN the patch is applied and no error is returned
//	THEN the cmd execution does not result in an error.
func TestExecPatchCmd(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	type args struct {
		chart     string
		version   string
		chartsDir string
		patchFile string
	}
	tests := []struct {
		name      string
		args      args
		hfs       fakes.FakeHelmChartFileSystem
		wantError error
	}{
		{
			name:      "testChartArgumentNilThrowsError",
			args:      args{chart: "", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagChartName, constants.FlagChartName, constants.FlagChartShorthand),
		},
		{
			name:      "testChartArgumentEmptyThrowsError",
			args:      args{chart: "\n", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagChartName),
		},
		{
			name:      "testVersionArgumentNilThrowsError",
			args:      args{chart: "chart", version: "", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagVersionName, constants.FlagVersionName, constants.FlagVersionShorthand),
		},
		{
			name:      "testVersionArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "\t", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagVersionName),
		},
		{
			name:      "testDirArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagDirName, constants.FlagDirName, constants.FlagDirShorthand),
		},
		{
			name:      "testDirArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "\n", patchFile: "/tmp/test.patch"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagDirName),
		},
		{
			name:      "testPatchFileArgumentNilThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: ""},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatMustSpecifyFlag, constants.FlagPatchFileName, constants.FlagPatchFileName, constants.FlagPatchFileShorthand),
		},
		{
			name:      "testPatchFileArgumentEmptyThrowsError",
			args:      args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: "\n"},
			hfs:       fakes.FakeHelmChartFileSystem{},
			wantError: fmt.Errorf(helpers.ErrFormatNotEmpty, constants.FlagPatchFileName),
		},
		{
			name: "testApplyPatchPatchThrowsError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeApplyPatchFile: func(chartsDir string, vzHelper vzhelpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
					return false, fmt.Errorf(vcmtesthelpers.DummyError)
				},
			},
			wantError: fmt.Errorf(vcmtesthelpers.DummyError),
		},
		{
			name: "testApplyPatchRetursNoError",
			args: args{chart: "chart", version: "0.0.1", chartsDir: "/tmp/charts", patchFile: "/tmp/test.patch"},
			hfs: fakes.FakeHelmChartFileSystem{
				FakeApplyPatchFile: func(chartsDir string, vzHelper vzhelpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
					return false, nil
				},
			},
			wantError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdPatch(rc, tt.hfs)
			cmd.PersistentFlags().Set(constants.FlagChartName, tt.args.chart)
			cmd.PersistentFlags().Set(constants.FlagVersionName, tt.args.version)
			cmd.PersistentFlags().Set(constants.FlagDirName, tt.args.chartsDir)
			cmd.PersistentFlags().Set(constants.FlagPatchFileName, tt.args.patchFile)
			err := cmd.Execute()
			if err != nil && tt.wantError == nil {
				t.Errorf("patch exec with args %v resulted in error %v", tt.args, err)
			}

			if err != nil && tt.wantError != nil && err.Error() != tt.wantError.Error() {
				t.Errorf("patch exec with args %v resulted in error %v, expected %v", tt.args, err, tt.wantError)
			}

			if err == nil && tt.wantError != nil {
				t.Errorf("patch exec with args %v resulted in no error, expected %v", tt.args, tt.wantError)
			}
		})
	}
}
