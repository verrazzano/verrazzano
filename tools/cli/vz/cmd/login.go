// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var username string
var password string

func init() {
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "Your Verrazzano username")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Your Verrazzano password")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login [Verrazzano API Server address]",
	Short: "Login to a Verrazzano environment",
	Long:  "Login to a Verrazzano environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := login(args); err != nil {
			return err
		}
		return nil
	},
}

func login(args []string) error {
	fmt.Println("login...")
	fmt.Printf(`  api server address: %s
  username: %s
  password: %s
`, args[0], username, password)
	return nil
}
