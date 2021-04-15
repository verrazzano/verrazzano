// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clustersCmd)
}

var clustersCmd = &cobra.Command{
	Use: "clusters",
	Short: "Information about clusters",
	Long: "Information about clusters",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := clusters(args); err != nil {
			return err
		}
		return nil
	},
}

func clusters(args []string) error {
	fmt.Println("clusters...")
	return nil
}