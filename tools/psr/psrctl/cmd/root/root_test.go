// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/explain"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/list"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/start"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/stop"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/update"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRootCmd(t *testing.T) {

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rootCmd := NewRootCmd(rc)
	assert.NotNil(t, rootCmd)

	// Verify the expected commands are defined
	assert.Len(t, rootCmd.Commands(), 6)
	foundCount := 0
	for _, cmd := range rootCmd.Commands() {
		switch cmd.Name() {
		case explain.CommandName:
			foundCount++
		case list.CommandName:
			foundCount++
		case start.CommandName:
			foundCount++
		case stop.CommandName:
			foundCount++
		case update.CommandName:
			foundCount++
		case version.CommandName:
			foundCount++
		}
	}
	assert.Equal(t, 6, foundCount)

	//// Verify help has the expected elements
	rootCmd.SetArgs([]string{fmt.Sprintf("--%s", constants.GlobalFlagHelp)})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "Usage:")
	assert.Contains(t, result, "Available Commands:")
	assert.Contains(t, result, "Flags:")
}
