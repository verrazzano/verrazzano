// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/yaml"
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

var apiExclusionList = []string{"pods", "replicasets", "endpoints", "endpointslices"}

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
		// Parse the group/version
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return err
		}
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 || !strings.Contains(resource.Verbs.String(), "list") {
				continue
			}
			// Skip items contained on the exclusion list
			if slices.Contains(apiExclusionList, resource.Name) {
				continue
			}

			gvr := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			if err = exportResource(dynamicClient, vzHelper, resource, gvr, namespace, appName); err != nil {
				return err
			}
		}
	}

	return nil
}

func exportResource(client dynamic.Interface, vzHelper helpers.VZHelper, resource metav1.APIResource, gvr schema.GroupVersionResource, namespace string, appName string) error {
	list, err := client.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to list GVR %s/%s/%s: %v", gvr.Group, gvr.Version, resource.Name, err)
	}

	// Export each resource that matches the OAM filters
	for _, item := range list.Items {
		// Skip items that do not match the OAM filtering rules
		if gvr.Group == "oam.verrazzano.io" {
			continue
		}
		labels := item.GetLabels()
		if labels["app.oam.dev/name"] != appName {
			continue
		}

		// Strip out some of the runtime information
		annotations := item.GetAnnotations()
		if annotations != nil {
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			item.SetAnnotations(annotations)
		}
		itemContent := item.UnstructuredContent()
		item.UnstructuredContent()
		mc := item.UnstructuredContent()["metadata"].(map[string]interface{})
		delete(mc, "creationTimestamp")
		delete(mc, "generateName")
		delete(mc, "managedFields")
		delete(mc, "ownerReferences")
		delete(mc, "resourceVersion")
		delete(mc, "uid")
		item.UnstructuredContent()["metadata"] = mc
		delete(itemContent, "status")

		// Marshall into yaml format and output
		yamlBytes, _ := yaml.Marshal(itemContent)
		fmt.Fprintf(vzHelper.GetOutputStream(), "%s\n---\n", yamlBytes)
	}
	return nil
}
