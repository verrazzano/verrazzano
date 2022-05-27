// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// NewCommand - utility method to create cobra commands
func NewCommand(vzHelper helpers.VZHelper, use string, short string, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}

	// Configure the IO streams
	cmd.SetOut(vzHelper.GetOutputStream())
	cmd.SetErr(vzHelper.GetErrorStream())
	cmd.SetIn(vzHelper.GetInputStream())

	// Disable usage output on errors
	cmd.SilenceUsage = true
	return cmd
}
