// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	name string
)

func init() {
	clusterRegisterCmd.Flags().StringVarP(&username, "name", "n", "", "Cluster name")
	clusterCmd.AddCommand(clusterRegisterCmd)
}

var clusterRegisterCmd = &cobra.Command{
	Use: "register",
	Short: "Register a cluster",
	Long: "Register a cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := register(args); err != nil {
			return err
		}
		return nil
	},
}

func register(args []string) error {
	fmt.Println("register cluster...")
	fmt.Printf("  name: %s\n", name)
	return nil
}