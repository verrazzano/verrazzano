// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
)

var kubeconfig string
var context string

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vz",
		Short: "Verrazzano CLI",
		Long:  "Verrazzano CLI",
	}

	// Add global flags
	cmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "c", "", "Kubernetes configuration file")
	cmd.PersistentFlags().StringVar(&context, "context", "", "The name of the kubeconfig context to use")

	cmd.AddCommand(status.NewCmdStatus())
	cmd.AddCommand(version.NewCmdVersion())

	return cmd
}
