// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/capi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestClusterArgs(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
		args       []string
		expectErr  bool
	}{
		{"cluster create with default name", createSubCommandName, []string{"--type", capi.NoClusterType}, false},
		{"cluster create with custom name and type", createSubCommandName, []string{"--name", "mycluster", "--type", capi.NoClusterType}, false},
		{"cluster create with custom name, type and image", createSubCommandName, []string{"--name", "mycluster", "--type", capi.NoClusterType, "--image", "somerepo.io/someimage"}, false},
		{"cluster create with unsupported type", createSubCommandName, []string{"--type", "unknown"}, true},
		{"cluster create with unknown flag", createSubCommandName, []string{"--unknown", "value"}, true},
		{"cluster with nonexistent subcommand", "nonexistent", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send stdout stderr to a byte buffer
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)
			rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
			cmd := NewCmdCluster(rc)
			args := []string{
				tt.subcommand,
			}
			args = append(args, tt.args...)
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tt.expectErr {
				asserts.Error(t, err)
				return
			} else {
				asserts.NoError(t, err)
			}
		})
	}
}

func TestClusterCreateHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	cmd := NewCmdCluster(rc)
	args := []string{createSubCommandName, "-h"}
	cmd.SetArgs(args)
	err := cmd.Execute()
	asserts.Nil(t, err)
	output := buf.String()
	// The image flag should be hidden in help
	asserts.NotContains(t, output, constants.ClusterImageFlagName)
	asserts.NotContains(t, output, constants.ClusterImageFlagHelp)

	asserts.Contains(t, output, constants.ClusterNameFlagName)
	asserts.Contains(t, output, constants.ClusterNameFlagHelp)

	asserts.Contains(t, output, constants.ClusterTypeFlagName)
	asserts.Contains(t, output, constants.ClusterTypeFlagHelp)
	fmt.Println(output)
}
