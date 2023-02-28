// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/analyze"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/cluster"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/tools/vz/cmd/install"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/uninstall"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/upgrade"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRootCmd(t *testing.T) {

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rootCmd := NewRootCmd(rc)
	assert.NotNil(t, rootCmd)

	// Verify the expected commands are defined
	assert.Len(t, rootCmd.Commands(), 8)
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
		case cluster.CommandName:
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
	assert.True(t, strings.Contains(buf.String(), "Usage:"))
	assert.True(t, strings.Contains(buf.String(), "Available Commands:"))
	assert.True(t, strings.Contains(buf.String(), "Flags:"))
}

func TestClusterCommandIsHiddenInHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rootCmd := NewRootCmd(rc)
	assert.NotNil(t, rootCmd)
	args := []string{"-h"}
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	assert.Nil(t, err)
	output := buf.String()

	// The create command should be hidden in VZ CLI help
	assert.NotContains(t, output, cluster.HelpShort)
}

// TestInstallCmdDefaultTimeout
// GIVEN a CLI install command with all defaults and --timeout=2s
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command times out
func TestInstallCmdDefaultTimeout(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(testhelpers.CreateTestVPOObjects()...).Build()
	cmd, buf, errBuf, _ := createNewTestCommandAndBuffers(t, c)
	cmdInstall, err := cmdhelpers.GetSubCommandInstall(cmd)

	cmdInstall.PersistentFlags().Set(constants.TimeoutFlag, "2s")
	cmdhelpers.SetDeleteFunc(cmdhelpers.FakeDeleteFunc)
	defer cmdhelpers.SetDefaultDeleteFunc()

	// Run install command
	err = cmdInstall.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Error: Timeout 2s exceeded waiting for install to complete\n", errBuf.String())
	assert.Contains(t, buf.String(), "Installing Verrazzano version v1.3.1")
}

func createNewTestCommandAndBuffers(t *testing.T, c client.Client) (*cobra.Command, *bytes.Buffer, *bytes.Buffer, *testhelpers.FakeRootCmdContext) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	if c != nil {
		rc.SetClient(c)
	}
	cmd := NewRootCmd(rc)
	assert.NotNil(t, cmd)
	return cmd, buf, errBuf, rc
}
