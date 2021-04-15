// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logoutCmd)
}

var logoutCmd = &cobra.Command{
	Use: "logout",
	Short: "Logout from a Verrazzano environment",
	Long: "Logout from a Verrazzano environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logout(args); err != nil {
			return err
		}
		return nil
	},
}

func logout(args []string) error {
	fmt.Println("logout...")
	return nil
}