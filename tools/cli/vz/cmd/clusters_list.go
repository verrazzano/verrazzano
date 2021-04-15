// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	clustersCmd.AddCommand(clusterListCmd)
}

var clusterListCmd = &cobra.Command{
	Use: "list",
	Short: "List the clusters",
	Long: "List the clusters",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := list(args); err != nil {
			return err
		}
		return nil
	},
}

func list(args []string) error {
	fmt.Println("list clusters...")
	return nil
}