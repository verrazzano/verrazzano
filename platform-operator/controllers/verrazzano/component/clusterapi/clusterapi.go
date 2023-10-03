// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const clusterctlYamlTemplate = `
{{- if .IncludeImagesHeader }}
images:
  {{- if not .ClusterAPIOverridesExists }}
  cluster-api:
    repository: {{.GetClusterAPIRepository}}
    tag: {{.GetClusterAPITag}}
  {{ end }}

  {{- if not .OCIOverridesExists }}
  infrastructure-oci:
    repository: {{.GetOCIRepository}}
    tag: {{.GetOCITag}}
  {{ end }}

  {{- if not .OCNEBootstrapOverridesExists }}
  bootstrap-ocne:
    repository: {{.GetOCNEBootstrapRepository}}
    tag: {{.GetOCNEBootstrapTag}}
  {{ end }}

  {{- if not .OCNEControlPlaneOverridesExists }}
  control-plane-ocne:
    repository: {{.GetOCNEControlPlaneRepository}}
    tag: {{.GetOCNEControlPlaneTag}}
  {{ end }}

  {{- if not .GetVerrazzanoAddonOverridesVersion }}
  addon-verrazzano:
    repository: {{.GetVerrazzanoAddonRepository}}
    tag: {{.GetVerrazzanoAddonTag}}
  {{ end }}
{{ end }}

providers:
  - name: "cluster-api"
    url: "{{.GetClusterAPIURL}}"
    type: "CoreProvider"
  - name: "oci"
    url: "{{.GetOCIURL}}"
    type: "InfrastructureProvider"
  - name: "ocne"
    url: "{{.GetOCNEBootstrapURL}}"
    type: "BootstrapProvider"
  - name: "ocne"
    url: "{{.GetOCNEControlPlaneURL}}"
    type: "ControlPlaneProvider"
  - name: "verrazzano"
    url: "{{.GetVerrazzanoAddonURL}}"
    type: "AddonProvider"
`

const (
	clusterTopology                           = "CLUSTER_TOPOLOGY"
	expClusterResourceSet                     = "EXP_CLUSTER_RESOURCE_SET"
	expMachinePool                            = "EXP_MACHINE_POOL"
	initOCIClientsOnStartup                   = "INIT_OCI_CLIENTS_ON_STARTUP"
	clusterAPIControllerImage                 = "cluster-api-controller"
	clusterAPIOCIControllerImage              = "cluster-api-oci-controller"
	clusterAPIOCNEBoostrapControllerImage     = "cluster-api-ocne-bootstrap-controller"
	clusterAPIOCNEControlPLaneControllerImage = "cluster-api-ocne-control-plane-controller"
	clusterAPIVerrazzanoAddonControllerImage  = "cluster-api-verrazzano-addon-controller"
	defaultClusterAPIDir                      = "/verrazzano/.cluster-api"
	clusterAPIDirEnv                          = "VERRAZZANO_CLUSTER_API_DIR"
	providerLabel                             = "cluster.x-k8s.io/provider"
	clusterAPIProvider                        = "cluster-api"
	bootstrapOcneProvider                     = "bootstrap-ocne"
	controlPlaneOcneProvider                  = "control-plane-ocne"
	infrastructureOciProvider                 = "infrastructure-oci"
	verrazzanoAddonProvider                   = "addon-verrazzano"
)

type ImageConfig struct {
	Version    string
	Repository string
	Tag        string
}

// PodMatcherClusterAPI matches pods with an out of date CoreProvider, BootstrapProvider, ControlPlaneProvider, or InfrastructureProvider.
type PodMatcherClusterAPI struct {
	coreProvider           string
	bootstrapProvider      string
	controlPlaneProvider   string
	infrastructureProvider string
	addonProvider          string
}

var clusterAPIDir = defaultClusterAPIDir
var providerGVR = schema.GroupVersionResource{
	Group:    "clusterctl.cluster.x-k8s.io",
	Version:  "v1alpha3",
	Resource: "providers",
}

// Functions needed for unit testing to set and reset .cluster-api directory
func setClusterAPIDir(dir string) {
	clusterAPIDir = dir
}
func resetClusterAPIDir() {
	clusterAPIDir = defaultClusterAPIDir
}

func getClusterAPIDir() string {
	if capiDir := os.Getenv(clusterAPIDirEnv); len(capiDir) > 0 {
		return capiDir
	}
	return clusterAPIDir
}

// preInstall implementation for the ClusterAPI Component
func preInstall(ctx spi.ComponentContext) error {
	err := setEnvVariables()
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to set environment variables needed by cluster-api providers: %v", err)
		return err
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return createClusterctlYaml(ctx)
}

// preUpgrade implementation for the ClusterAPI Component
func preUpgrade(ctx spi.ComponentContext) error {
	err := setEnvVariables()
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to set environment variables needed by cluster-api providers: %v", err)
		return err
	}

	// Create the clusterctl.yaml used when upgrading CAPI.
	return createClusterctlYaml(ctx)
}

// setEnvVariables sets the environment variables needed for CAPI providers.
func setEnvVariables() error {
	// Startup the OCI infrastructure provider without requiring OCI credentials
	err := os.Setenv(initOCIClientsOnStartup, "false")
	if err != nil {
		return err
	}

	// Enable experimental feature cluster resource set at boot up
	err = os.Setenv(expClusterResourceSet, "true")
	if err != nil {
		return err
	}

	// Enable experimental feature machine pool at boot up
	err = os.Setenv(expMachinePool, "true")
	if err != nil {
		return err
	}

	// Enable experimental feature cluster topology at boot up
	return os.Setenv(clusterTopology, "true")
}

// createClusterctlYaml creates clusterctl.yaml with image overrides from the Verrazzano BOM
func createClusterctlYaml(ctx spi.ComponentContext) error {
	// Get the image overrides and versions for the CAPI images.
	overrides, err := createOverrides(ctx)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to create image overrides: %v", err)
		return err
	}

	// Apply the image overrides and versions to generate clusterctl.yaml
	result, err := applyTemplate(ctx, clusterctlYamlTemplate, newOverridesContext(overrides))
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to apply template for creating clusterctl.yaml: %v", err)
		return err
	}

	err = os.Mkdir(getClusterAPIDir(), 0755)
	if err != nil {
		if !os.IsExist(err) {
			ctx.Log().ErrorfThrottled("Failed to create directory %s: %v", getClusterAPIDir(), err)
			return err
		}
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	err = os.WriteFile(getClusterAPIDir()+"/clusterctl.yaml", result.Bytes(), 0600)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to create file %s: %v", getClusterAPIDir()+"/clusterctl.yaml", err)
	}

	return err
}

// applyTemplate applies the CAPI provider image overrides and versions to the clusterctl.yaml template
func applyTemplate(ctx spi.ComponentContext, templateContent string, params interface{}) (bytes.Buffer, error) {
	// Parse the template file
	capiYaml, err := template.New("capi").Parse(templateContent)
	if err != nil {
		ctx.Log().ErrorfThrottled("template error: %v", err)
		return bytes.Buffer{}, err
	}

	// Apply the replacement parameters to the template
	var buf bytes.Buffer
	err = capiYaml.Execute(&buf, &params)
	if err != nil {
		ctx.Log().ErrorfThrottled("template execution error: %v", err)
		return bytes.Buffer{}, err
	}

	// Return the result containing the processed template
	return buf, nil
}

// Initializes Cluster API controller image versions considering overrides from VZ CR
func (c *PodMatcherClusterAPI) initializeImageVersionsOverrides(log vzlog.VerrazzanoLogger, overrides OverridesInterface) error {
	c.coreProvider = overrides.GetClusterAPIControllerFullImagePath()
	c.infrastructureProvider = overrides.GetOCIControllerFullImagePath()
	c.bootstrapProvider = overrides.GetOCNEBootstrapControllerFullImagePath()
	c.controlPlaneProvider = overrides.GetOCNEControlPlaneControllerFullImagePath()
	c.addonProvider = overrides.GetVerrazzanoAddonControllerFullImagePath()
	return nil
}

// applyUpgradeVersion returns true and corresponding version if either URL overrides, version overrides or bom version is specified. Otherwise, return false and empty version.
func applyUpgradeVersion(log vzlog.VerrazzanoLogger, URLOverrides, versionOverrides, bomVersion, imageName, actualImage, expectedImage string) (bool, string) {
	if len(URLOverrides) > 0 {
		// Version is expected as the penultimate component of the overrides URL
		splitURL := strings.Split(URLOverrides, "/")
		version := splitURL[len(splitURL)-2]
		log.Infof("URL Overrides version \"%v\" extracted from URL Overriddes %v", version, URLOverrides)
		return true, version
	}
	if len(versionOverrides) > 0 {
		return true, versionOverrides
	}
	if strings.Contains(actualImage, imageName) && actualImage != expectedImage {
		log.Infof("Image %v is out of date, actual image: %v, expected image: %v", imageName, actualImage, expectedImage)
		return true, bomVersion
	}
	return false, ""
}

// MatchAndPrepareUpgradeOptions when a pod has an outdated cluster api controllers images and prepares upgrade options for outdated images.
func (c *PodMatcherClusterAPI) matchAndPrepareUpgradeOptions(ctx spi.ComponentContext, overrides OverridesInterface) (capiUpgradeOptions, error) {
	c.initializeImageVersionsOverrides(ctx.Log(), overrides)
	applyUpgradeOptions := capiUpgradeOptions{}
	podList := &v1.PodList{}
	if err := ctx.Client().List(context.TODO(), podList, &client.ListOptions{Namespace: ComponentNamespace}); err != nil {
		return applyUpgradeOptions, err
	}

	const formatString = "%s/%s:%s"
	for _, pod := range podList.Items {
		for _, co := range pod.Spec.Containers {
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetClusterAPIOverridesURL(), overrides.GetClusterAPIOverridesVersion(), overrides.GetClusterAPIBomVersion(), clusterAPIControllerImage, co.Image, c.coreProvider); ok {
				applyUpgradeOptions.CoreProvider = fmt.Sprintf(formatString, ComponentNamespace, clusterAPIProviderName, version)
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCNEBootstrapOverridesURL(), overrides.GetOCNEBootstrapOverridesVersion(), overrides.GetOCNEBootstrapBomVersion(), clusterAPIOCNEBoostrapControllerImage, co.Image, c.bootstrapProvider); ok {
				applyUpgradeOptions.BootstrapProviders = append(applyUpgradeOptions.BootstrapProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, version))
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCNEControlPlaneOverridesURL(), overrides.GetOCNEControlPlaneOverridesVersion(), overrides.GetOCNEControlPlaneBomVersion(), clusterAPIOCNEControlPLaneControllerImage, co.Image, c.controlPlaneProvider); ok {
				applyUpgradeOptions.ControlPlaneProviders = append(applyUpgradeOptions.ControlPlaneProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, version))
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCIOverridesURL(), overrides.GetOCIOverridesVersion(), overrides.GetOCIBomVersion(), clusterAPIOCIControllerImage, co.Image, c.infrastructureProvider); ok {
				applyUpgradeOptions.InfrastructureProviders = append(applyUpgradeOptions.InfrastructureProviders, fmt.Sprintf(formatString, ComponentNamespace, ociProviderName, version))
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetVerrazzanoAddonOverridesURL(), overrides.GetVerrazzanoAddonOverridesVersion(), overrides.GetVerrazzanoAddonBomVersion(), clusterAPIVerrazzanoAddonControllerImage, co.Image, c.addonProvider); ok {
				applyUpgradeOptions.AddonProviders = append(applyUpgradeOptions.AddonProviders, fmt.Sprintf(formatString, ComponentNamespace, verrazzanoAddonProviderName, version))
			}
		}
	}

	return applyUpgradeOptions, nil
}

// isUpgradeOptionsNotEmpty returns true if any of the options is not empty
func isUpgradeOptionsNotEmpty(upgradeOptions capiUpgradeOptions) bool {
	return len(upgradeOptions.CoreProvider) != 0 ||
		len(upgradeOptions.BootstrapProviders) != 0 ||
		len(upgradeOptions.ControlPlaneProviders) != 0 ||
		len(upgradeOptions.InfrastructureProviders) != 0 ||
		len(upgradeOptions.AddonProviders) != 0
}

func getComponentsToUpgrade(c client.Client, options capiUpgradeOptions) ([]client.Object, error) {
	var componentObjects []client.Object

	if options.CoreProvider != "" {
		coreComponents, err := getComponentsForProviderType(c, clusterAPIProvider, constants.VerrazzanoCAPINamespace)
		if err != nil {
			return componentObjects, err
		}
		componentObjects = append(componentObjects, coreComponents...)
	}

	if len(options.BootstrapProviders) != 0 {
		boostrapComponents, err := getComponentsForProviderType(c, bootstrapOcneProvider, constants.VerrazzanoCAPINamespace)
		if err != nil {
			return componentObjects, err
		}
		componentObjects = append(componentObjects, boostrapComponents...)
	}

	if len(options.ControlPlaneProviders) != 0 {
		controlPlaneComponents, err := getComponentsForProviderType(c, controlPlaneOcneProvider, constants.VerrazzanoCAPINamespace)
		if err != nil {
			return componentObjects, err
		}
		componentObjects = append(componentObjects, controlPlaneComponents...)
	}

	if len(options.InfrastructureProviders) != 0 {
		infrastructureComponents, err := getComponentsForProviderType(c, infrastructureOciProvider, constants.VerrazzanoCAPINamespace)
		if err != nil {
			return componentObjects, err
		}
		componentObjects = append(componentObjects, infrastructureComponents...)
	}

	if len(options.AddonProviders) != 0 {
		addonComponents, err := getComponentsForProviderType(c, verrazzanoAddonProvider, constants.VerrazzanoCAPINamespace)
		if err != nil {
			return componentObjects, err
		}
		componentObjects = append(componentObjects, addonComponents...)
	}
	return componentObjects, nil
}

// getComponentsForProviderType - return a list of ClusterRoles, ClusterRoleBindings, Roles and RoleBindings that are associated with provider specified.
func getComponentsForProviderType(c client.Client, providerName string, namespace string) ([]client.Object, error) {
	var objs []client.Object

	// ClusterRoles
	clusterRoles := &rbac.ClusterRoleList{}
	if err := c.List(context.TODO(), clusterRoles, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{providerLabel: providerName})}); err != nil {
		return objs, err
	}
	for i := range clusterRoles.Items {
		objs = append(objs, &clusterRoles.Items[i])
	}

	// ClusterRoleBindings
	clusterRoleBindings := &rbac.ClusterRoleBindingList{}
	if err := c.List(context.TODO(), clusterRoleBindings, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{providerLabel: providerName})}); err != nil {
		return objs, err
	}
	for i := range clusterRoleBindings.Items {
		objs = append(objs, &clusterRoleBindings.Items[i])
	}

	// Roles
	roles := &rbac.RoleList{}
	if err := c.List(context.TODO(), roles, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{providerLabel: providerName}),
	}); err != nil {
		return objs, err
	}
	for i := range roles.Items {
		objs = append(objs, &roles.Items[i])
	}

	// RoleBindings
	roleBindings := &rbac.RoleBindingList{}
	if err := c.List(context.TODO(), roleBindings, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{providerLabel: providerName}),
	}); err != nil {
		return objs, err
	}
	for i := range roleBindings.Items {
		objs = append(objs, &roleBindings.Items[i])
	}

	return objs, nil
}

// checkClusterAPIDeployment checks for the existence of a deployment for a cluster api provider
func checkClusterAPIDeployment(ctx spi.ComponentContext, deploymentName string) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: deploymentName}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to get %s/%s deployment: %v", ComponentNamespace, deploymentName, err)
		return false, err
	}
	return true, nil
}
