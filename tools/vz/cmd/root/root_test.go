// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/analyze"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/export"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/uninstall"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/upgrade"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
)

func TestNewRootCmd(t *testing.T) {

	rc := helpers.NewFakeRootCmdContextWithFiles(t)
	defer helpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	rootCmd := NewRootCmd(rc)
	assert.NotNil(t, rootCmd)

	// Verify the expected commands are defined
	assert.Len(t, rootCmd.Commands(), 9)
	foundCount := 0
	for _, cmd := range rootCmd.Commands() {
		switch cmd.Name() {
		case status.CommandName:
			foundCount++
		case version.CommandName:
			foundCount++
		case install.CommandName:
			foundCount++
		case upgrade.CommandName:
			foundCount++
		case uninstall.CommandName:
			foundCount++
		case analyze.CommandName:
			foundCount++
		case bugreport.CommandName:
			foundCount++
		case export.CommandName:
			foundCount++
		}
	}
	assert.Equal(t, 8, foundCount)

	// Verify the expected global flags are defined
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup(constants.GlobalFlagKubeConfig))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup(constants.GlobalFlagContext))

	// Verify help has the expected elements
	rootCmd.SetArgs([]string{fmt.Sprintf("--%s", constants.GlobalFlagHelp)})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	fileBuffer, err := os.ReadFile(rc.Out.Name())
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(fileBuffer), "Usage:"))
	assert.True(t, strings.Contains(string(fileBuffer), "Available Commands:"))
	assert.True(t, strings.Contains(string(fileBuffer), "Flags:"))
}
