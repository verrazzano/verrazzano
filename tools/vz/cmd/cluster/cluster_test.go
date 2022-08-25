// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"bytes"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/capi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const nameFlag = "--" + constants.ClusterNameFlagName
const typeFlag = "--" + constants.ClusterTypeFlagName

func TestClusterArgs(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
		args       []string
		expectErr  bool
	}{
		// capi.NoClusterType is used in all testing so that we don't trigger an actual cluster create/delete
		{"cluster create with default name", createSubCommandName, []string{typeFlag, capi.NoClusterType}, false},
		{"cluster create with custom name and type", createSubCommandName, []string{nameFlag, "mycluster", typeFlag, capi.NoClusterType}, false},
		{"cluster create with custom name, type and image", createSubCommandName, []string{nameFlag, "mycluster", typeFlag, capi.NoClusterType, "--image", "somerepo.io/someimage"}, false},
		{"cluster create with unsupported type", createSubCommandName, []string{typeFlag, "unknown"}, true},
		{"cluster create with unknown flag", createSubCommandName, []string{"--unknown", "value"}, true},
		{"cluster delete with default name", deleteSubCommandName, []string{typeFlag, capi.NoClusterType}, false},
		{"cluster delete with custom name", deleteSubCommandName, []string{nameFlag, "randomcluster", typeFlag, capi.NoClusterType}, false},
		{"cluster delete with unknown flag", deleteSubCommandName, []string{"--someflag", "randomcluster"}, true},
		{"cluster with nonexistent subcommand", "nonexistent", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(tt.subcommand, tt.args)
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func TestClusterCreateHelp(t *testing.T) {
	output, _, err := runCommand(createSubCommandName, []string{"-h"})
	asserts.Nil(t, err)
	// The image flag should be hidden in help
	asserts.NotContains(t, output, constants.ClusterImageFlagName)
	asserts.NotContains(t, output, constants.ClusterImageFlagHelp)

	asserts.Contains(t, output, constants.ClusterNameFlagName)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)

	asserts.Contains(t, output, constants.ClusterTypeFlagName)
	asserts.Contains(t, output, constants.ClusterTypeFlagHelp)
}

func TestClusterDeleteHelp(t *testing.T) {
	output, _, err := runCommand(deleteSubCommandName, []string{"-h"})
	asserts.Nil(t, err)
	asserts.Contains(t, output, constants.ClusterNameFlagName)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)

	// The type flag should be hidden in delete help
	asserts.NotContains(t, output, constants.ClusterTypeFlagName)
	asserts.NotContains(t, output, constants.ClusterTypeFlagHelp)
}

func runCommand(subCommand string, args []string) (string, string, error) {
	// Send stdout and stderr to byte buffers
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdCluster(rc)
	subCommandArr := []string{subCommand}
	cmd.SetArgs(append(subCommandArr, args...))
	err := cmd.Execute()
	return buf.String(), errBuf.String(), err
}
