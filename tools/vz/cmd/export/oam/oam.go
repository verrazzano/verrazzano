// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/dynamic"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"k8s.io/utils/strings/slices"

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

// apiIgnoreList - list of APIs to ignore for export
var apiIgnoreList = []string{"events", "pods", "poddisruptionbudgets", "localsubjectaccessreviews", "controllerrevisions", "csistoragecapacities", "leases"}

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
	appName, err := cmd.PersistentFlags().GetString(constants.AppNameFlag)
	if err != nil {
		return fmt.Errorf(flagErrorStr, err.Error())
	}
	if len(appName) == 0 {
		return fmt.Errorf("A value for --%s is required", constants.AppNameFlag)
	}

	// Get the namespace
	namespace, err := cmd.PersistentFlags().GetString(constants.NamespaceFlag)
	if err != nil {
		return fmt.Errorf(flagErrorStr, err.Error())
	}

	// Get the dynamic client
	dynamicClient, err := vzHelper.GetDynamicClient(cmd)
	if err != nil {
		return err
	}

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
			// Skip the following resource types
			if slices.Contains(apiIgnoreList, resource.Name) {
				continue
			}
			if err = exportResource(dynamicClient, vzHelper, resource, namespace, appName); err != nil {
				return err
			}
		}
	}

	return nil
}

func exportResource(client dynamic.Interface, vzHelper helpers.VZHelper, resource metav1.APIResource, namespace string, appName string) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), "Processing resource %q in namespace %q for OAM app %q \n", resource.Name, namespace, appName)

	foo := schema.GroupVersionResource{Group: resource.Group, Version: resource.Version, Resource: resource.Name}
	list, err := client.Resource(foo).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Export each resource that matches the OAM filters
	for _, item := range list.Items {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Found item %s", item.GetName())
	}
	return nil
}
