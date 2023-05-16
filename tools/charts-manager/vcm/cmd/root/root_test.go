// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/diff"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/patch"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/pull"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/fakes"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
)

// TestNewRootCmd tests that function TestNewRootCmd creates root cmd with correct sub commands
// GIVEN a call to TestNewRootCmd
//
//	WHEN correct arguments are passed
//	THEN the root cmd instance created contains all the required flags.
func TestNewRootCmd(t *testing.T) {
	rc, cleanup, err := vcmtesthelpers.ContextSetup()
	assert.NoError(t, err)
	defer cleanup()
	cmd := NewRootCmd(rc, fakes.FakeHelmChartFileSystem{}, fakes.FakeHelmConfig{})
	assert.NotNil(t, cmd, "command is nil")
	pullCmdFound := false
	diffCmdFound := false
	patchCmdFound := false
	for _, command := range cmd.Commands() {
		if command.Name() == pull.CommandName {
			pullCmdFound = true
		}

		if command.Name() == diff.CommandName {
			diffCmdFound = true
		}

		if command.Name() == patch.CommandName {
			patchCmdFound = true
		}
	}
	assert.True(t, pullCmdFound, "pull command not added,")
	assert.True(t, diffCmdFound, "diff command not added,")
	assert.True(t, patchCmdFound, "patch command not added,")
}
