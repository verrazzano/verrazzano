// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	projectsCmd.AddCommand(projectListCmd)
}

var projectListCmd = &cobra.Command{
	Use: "list",
	Short: "List projects",
	Long: "List projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := listProjects(args); err != nil {
			return err
		}
		return nil
	},
}

func listProjects(args []string) error {
	fmt.Println("list projects...")
	return nil
}