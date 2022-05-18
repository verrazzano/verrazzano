// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"fmt"

	"github.com/spf13/cobra"
)

const CommandName = "status"

func NewCmdStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: "Status of the Verrazzano install and access endpoints",
		Long:  "Status of the Verrazzano install and access endpoints",
		Run:   runCmdStatus,
	}
	return cmd
}

func runCmdStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
