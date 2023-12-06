// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	flagErrorStr = "error fetching flag: %s"
	CommandName  = "oam"
	helpShort    = "Export OAM"
	helpLong     = `Export the standard Kubernetes definition of an OAM application.`
	helpExample  = `
TBD
`
)

func NewCmdExportOAM(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdExportOAM(cmd, vzHelper)
	}

	cmd.Example = helpExample

	cmd.PersistentFlags().String(constants.NamespaceFlag, constants.NamespaceFlagDefault, constants.NamespaceFlagUsage)
	cmd.PersistentFlags().String(constants.AppNameFlag, constants.AppNameFlagDefault, constants.AppNameFlagUsage)

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}

func RunCmdExportOAM(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	// Get the OAM application name
	name, err := cmd.PersistentFlags().GetString(constants.AppNameFlag)
	if err != nil {
		return fmt.Errorf(flagErrorStr, err.Error())
	}
	if len(name) == 0 {
		return fmt.Errorf("A value for --%s is required", constants.AppNameFlag)
	}

	// Get the namespace
	namespace, err := cmd.PersistentFlags().GetString(constants.NamespaceFlag)
	if err != nil {
		return fmt.Errorf(flagErrorStr, err.Error())
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), "Namespace = %q\n", namespace)

	// Get the controller runtime client
	/*
		client, err := vzHelper.GetClient(cmd)
		if err != nil {
			return err
		}
	*/

	// Get the dynamic client
	/*
		dynamicClient, err := vzHelper.GetDynamicClient(cmd)
		if err != nil {
			return err
		}
	*/

	// Get the list of API namespaced resources
	disco, err := vzHelper.GetDiscoveryClient(cmd)
	if err != nil {
		return err
	}
	lists, err := disco.ServerPreferredNamespacedResources()
	if err != nil {
		return err
	}

	for _, list := range lists {
		for _, resource := range list.APIResources {
			fmt.Fprintf(vzHelper.GetErrorStream(), "Resource: %q Group: %q Version: %q \n", resource.Name, resource.Group, resource.Version)
		}
	}

	return nil
}
