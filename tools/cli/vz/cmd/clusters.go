// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clustersCmd)
}

var clustersCmd = &cobra.Command{
	Use: "clusters",
	Short: "Information about clusters",
	Long: "Information about clusters",
}
