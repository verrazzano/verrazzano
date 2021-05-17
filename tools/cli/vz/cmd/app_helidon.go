// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	appCmd.AddCommand(helidonCmd)
}

var helidonCmd = &cobra.Command{
	Use:   "helidon",
	Short: "Work with Helidon applications",
	Long:  "Work with Helidon applications",
}
