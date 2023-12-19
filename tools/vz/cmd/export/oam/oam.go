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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

const (
	flagErrorStr     = "error fetching flag: %s"
	defaultNamespace = "default"
	CommandName      = "oam"
	helpShort        = "Export Kubernetes objects for an OAM application"
	helpLong         = `Export the standard Kubernetes objects that were generated for an OAM application`
	helpExample      = `
# Export the Kubernetes objects that were generated for the OAM application named hello-helidon
vz export oam --namespace hello-helidon --name hello-helidon > myapp.yaml
`
	groupVerrazzanoOAM = "oam.verrazzano.io"
	groupCoreOAM       = "core.oam.dev"
	versionV1Alpha1    = "v1alpha1"
	versionV1Alpha2    = "v1alpha2"
	specKey            = "spec"
	metadataKey        = "metadata"
	statusKey          = "status"
)

var metadataRuntimeKeys = []string{"creationTimestamp", "generation", "generateName", "managedFields", "ownerReferences", "resourceVersion", "uid", "finalizers"}
var serviceSpecRuntimeKeys = []string{"clusterIP", "clusterIPs"}

// excludedAPIResources map of API resources to always exclude (note this is not currently taking into account group/version)
var excludedAPIResources = map[string]bool{
	"pods":                     true,
	"replicasets":              true,
	"endpoints":                true,
	"endpointslices":           true,
	"controllerrevisions":      true,
	"events":                   true,
	"applicationconfiguration": true,
	"component":                true,
}

// includedAPIResources map of API resources to always include (note this is not currently taking into account group/version)
var includedAPIResources = map[string]bool{
	"servicemonitors": true,
}

var gvrIngressTrait = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "ingresstraits")
var gvrLoggingTrait = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "loggingtraits")
var gvrManualScalerTrait = gvrFor(groupCoreOAM, versionV1Alpha2, "manualscalertraits")
var gvrMetricsTrait = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "metricstraits")
var gvrCoherenceWorkload = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "verrazzanocoherenceworkloads")
var gvrHelidonWorkload = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "verrazzanohelidonworkloads")
var gvrWeblogicWorkload = gvrFor(groupVerrazzanoOAM, versionV1Alpha1, "verrazzanoweblogicworkloads")
var traitTypes = []schema.GroupVersionResource{
	gvrIngressTrait,
	gvrLoggingTrait,
	gvrManualScalerTrait,
	gvrMetricsTrait,
	gvrCoherenceWorkload,
	gvrHelidonWorkload,
	gvrWeblogicWorkload,
}

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

	ownerRefs, err := getOwnerRefs(dynamicClient, namespace, appName)
	if err != nil {
		return err
	}

	// Get the list of API namespaced resources
	disco, err := vzHelper.GetDiscoveryClient(cmd)
	if err != nil {
		return err
	}
	lists, err := disco.ServerPreferredResources()
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
			if excludedAPIResources[resource.Name] {
				continue
			}

			gvr := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			if err = exportResource(dynamicClient, vzHelper, resource, gvr, namespace, appName, ownerRefs); err != nil {
				return err
			}
		}
	}
	if err := exportTLSSecrets(dynamicClient, vzHelper, namespace, ownerRefs); err != nil {
		return err
	}

	return nil
}

func getOwnerRefs(client dynamic.Interface, namespace, appName string) (map[string]bool, error) {
	ownerResourceNames := map[string]bool{
		appName: true,
	}

	for _, traitGVR := range traitTypes {
		list, err := client.Resource(traitGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
				continue
			}
			return nil, err
		}
		for _, item := range list.Items {
			if isOAMAppLabel(item.GetLabels(), appName) {
				ownerResourceNames[item.GetName()] = true
			}
		}
	}
	return ownerResourceNames, nil
}

// exportResource - export a single, sanitized resource to the output stream
func exportResource(client dynamic.Interface, vzHelper helpers.VZHelper, resource metav1.APIResource, gvr schema.GroupVersionResource, namespace string, appName string, ownerRefs map[string]bool) error {
	// Cluster wide and namespaced resources are passed it.  Override the command line namespace to include cluster context objects.
	if namespace != defaultNamespace && !resource.Namespaced {
		namespace = defaultNamespace
	}
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
		if !includedAPIResources[resource.Name] {
			if gvr.Group == groupVerrazzanoOAM || gvr.Group == groupCoreOAM {
				continue
			}

			labels := item.GetLabels()
			isAppResource := isOAMAppLabel(labels, appName) || isOwned(item, ownerRefs) || isFluentdConfigMap(item)
			if !isAppResource {
				continue
			}
		}

		printSanitized(item, vzHelper)
	}
	return nil
}

func exportTLSSecrets(client dynamic.Interface, vzHelper helpers.VZHelper, namespace string, ownerRefs map[string]bool) error {
	gateways, err := client.Resource(gvrFor("networking.istio.io", "v1beta1", "gateways")).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	var ownedGateways []unstructured.Unstructured
	for _, gw := range gateways.Items {
		if isOwned(gw, ownerRefs) {
			ownedGateways = append(ownedGateways, gw)
		}
	}
	credentialNames := getGatewayCredentialNames(ownedGateways)

	secrets, err := client.Resource(gvrFor("", "v1", "secrets")).Namespace("istio-system").List(context.Background(), metav1.ListOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	for _, secret := range secrets.Items {
		if credentialNames[secret.GetName()] {
			printSanitized(secret, vzHelper)
		}
	}
	return nil
}

func getGatewayCredentialNames(gateways []unstructured.Unstructured) map[string]bool {
	credentialNames := map[string]bool{}
	for _, gw := range gateways {
		servers, found, err := unstructured.NestedSlice(gw.Object, "spec", "servers")
		if !found || err != nil {
			continue
		}
		for _, server := range servers {
			credentialName := getServerCredentialName(server)
			if credentialName != nil {
				credentialNames[*credentialName] = true
			}
		}
	}
	return credentialNames
}

func getServerCredentialName(server interface{}) *string {
	serverMap, ok := server.(map[string]interface{})
	if !ok {
		return nil
	}
	credentialName, found, err := unstructured.NestedString(serverMap, "tls", "credentialName")
	if !found || err != nil {
		return nil
	}
	return &credentialName
}

func isOAMAppLabel(labels map[string]string, appName string) bool {
	return labels != nil && labels["app.oam.dev/name"] == appName
}

func isOwned(item unstructured.Unstructured, ownerRefs map[string]bool) bool {
	for _, ownerRef := range item.GetOwnerReferences() {
		if ownerRefs[ownerRef.Name] {
			return true
		}
	}
	return false
}

func isFluentdConfigMap(item unstructured.Unstructured) bool {
	return item.GetKind() == "ConfigMap" && strings.HasPrefix(item.GetName(), "fluentd-config-")
}

func printSanitized(item unstructured.Unstructured, vzHelper helpers.VZHelper) {
	itemContent := sanitize(item)
	// Marshall into yaml format and output
	yamlBytes, _ := yaml.Marshal(itemContent)
	fmt.Fprintf(vzHelper.GetOutputStream(), "%s\n---\n", yamlBytes)
}

// sanitize removes runtime metadata from objects
func sanitize(item unstructured.Unstructured) map[string]interface{} {
	// Strip out some of the runtime information
	annotations := item.GetAnnotations()
	if annotations != nil {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		item.SetAnnotations(annotations)
	}
	itemContent := item.UnstructuredContent()
	item.UnstructuredContent()
	delete(itemContent, statusKey)
	deleteNestedKeys(itemContent, metadataKey, metadataRuntimeKeys)
	gvk := item.GroupVersionKind()
	switch gvk {
	case gvkFor("", "v1", "Service"):
		deleteNestedKeys(itemContent, specKey, serviceSpecRuntimeKeys)
	}
	return itemContent
}

func deleteNestedKeys(m map[string]interface{}, key string, toDelete []string) {
	n, ok := m[key].(map[string]interface{})
	if !ok {
		return
	}
	for _, k := range toDelete {
		delete(n, k)
	}
	m[key] = n
}

func gvrFor(group, version, resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
}

func gvkFor(group, version, kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
}

func listResource(client dynamic.Interface, resource metav1.APIResource, gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error) {
	if resource.Namespaced {
		return client.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	}
	return client.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
}
