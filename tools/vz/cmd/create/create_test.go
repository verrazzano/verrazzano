// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"bytes"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/capi"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestCreateSubCommands(t *testing.T) {
	tests := []struct {
		name        string
		subcommand  string
		args        []string
		expectedErr bool
	}{
		{"create cluster with default name", "cluster", []string{"--type", capi.NoClusterType}, false},
		{"create cluster with custom name and type", "cluster", []string{"--name", "mycluster", "--type", capi.NoClusterType}, false},
		{"create cluster with unsupported type", "cluster", []string{"--type", "unknown"}, true},
		{"create cluster with unknown flag", "cluster", []string{"--unknown", "value"}, true},
		{"create with nonexistent subcommand", "nonexistent", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send stdout stderr to a byte buffer
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)
			rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
			cmd := NewCmdCreate(rc)
			args := []string{
				tt.subcommand,
			}
			args = append(args, tt.args...)
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tt.expectedErr {
				asserts.Error(t, err)
				return
			} else {
				asserts.NoError(t, err)
			}
		})
	}
}
