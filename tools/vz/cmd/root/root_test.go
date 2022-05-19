// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRootCmd(t *testing.T) {

	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	rootCmd := NewRootCmd(rc)
	assert.NotNil(t, rootCmd)

	// Verify the expected commands are defined
	assert.Len(t, rootCmd.Commands(), 2)
	foundCount := 0
	for _, cmd := range rootCmd.Commands() {
		switch cmd.Name() {
		case status.CommandName:
			foundCount++
		case version.CommandName:
			foundCount++
		}
	}
	assert.Equal(t, 2, foundCount)

	// Verify the expected global flags are defined
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup(GlobalFlagKubeconfig))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup(GlobalFlagContext))
}
