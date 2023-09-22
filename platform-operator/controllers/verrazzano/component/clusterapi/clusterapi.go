// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

const rbacGroup = "rbac.authorization.k8s.io"

const clusterctlYamlTemplate = `
{{- if .IncludeImagesHeader }}
images:
  {{- if not .GetClusterAPIOverridesVersion }}
  cluster-api:
    repository: {{.GetClusterAPIRepository}}
    tag: {{.GetClusterAPITag}}
  {{ end }}

  {{- if not .GetOCIOverridesVersion }}
  infrastructure-oci:
    repository: {{.GetOCIRepository}}
    tag: {{.GetOCITag}}
  {{ end }}

  {{- if not .GetOCNEBootstrapOverridesVersion }}
  bootstrap-ocne:
    repository: {{.GetOCNEBootstrapRepository}}
    tag: {{.GetOCNEBootstrapTag}}
  {{ end }}

  {{- if not .GetOCNEControlPlaneOverridesVersion }}
  control-plane-ocne:
    repository: {{.GetOCNEControlPlaneRepository}}
    tag: {{.GetOCNEControlPlaneTag}}
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
`

const (
	clusterTopology                           = "CLUSTER_TOPOLOGY"
	expClusterResourceSet                     = "EXP_CLUSTER_RESOURCE_SET"
	expMachinePool                            = "EXP_MACHINE_POOL"
	initOCIClientsOnStartup                   = "INIT_OCI_CLIENTS_ON_STARTUP"
	goproxy                                   = "GOPROXY"
	clusterAPIControllerImage                 = "cluster-api-controller"
	clusterAPIOCIControllerImage              = "cluster-api-oci-controller"
	clusterAPIOCNEBoostrapControllerImage     = "cluster-api-ocne-bootstrap-controller"
	clusterAPIOCNEControlPLaneControllerImage = "cluster-api-ocne-control-plane-controller"
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
}

const (
	defaultClusterAPIDir = "/verrazzano/.cluster-api"
	clusterAPIDirEnv     = "VERRAZZANO_CLUSTER_API_DIR"
)

var clusterAPIDir = defaultClusterAPIDir

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
		return err
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return createClusterctlYaml(ctx)
}

// preUpgrade implementation for the ClusterAPI Component
func preUpgrade(ctx spi.ComponentContext) error {
	err := setEnvVariables()
	if err != nil {
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
		return err
	}

	// Apply the image overrides and versions to generate clusterctl.yaml
	result, err := applyTemplate(clusterctlYamlTemplate, newOverridesContext(overrides))
	if err != nil {
		return err
	}

	err = os.Mkdir(getClusterAPIDir(), 0755)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return os.WriteFile(getClusterAPIDir()+"/clusterctl.yaml", result.Bytes(), 0600)
}

// applyTemplate applies the CAPI provider image overrides and versions to the clusterctl.yaml template
func applyTemplate(templateContent string, params interface{}) (bytes.Buffer, error) {
	// Parse the template file
	capiYaml, err := template.New("capi").Parse(templateContent)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Apply the replacement parameters to the template
	var buf bytes.Buffer
	err = capiYaml.Execute(&buf, &params)
	if err != nil {
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
	return nil
}

// applyUpgradeVersion returns true and corresponding version if either version overrides or bom version is specified. Otherwise, return false and empty version.
func applyUpgradeVersion(log vzlog.VerrazzanoLogger, versionOverrides, bomVersion, imageName, actualImage, expectedImage string) (bool, string) {
	if len(versionOverrides) > 0 {
		return true, versionOverrides
	}
	if strings.Contains(actualImage, imageName) && actualImage != expectedImage {
		log.Infof("Image %v is out of date,  actual image: %v, expected image: %v", imageName, actualImage, expectedImage)
		return true, bomVersion
	}
	return false, ""
}

// MatchAndPrepareUpgradeOptions when a pod has an outdated cluster api controllers images and prepares upgrade options for outdated images.
func (c *PodMatcherClusterAPI) matchAndPrepareUpgradeOptions(ctx spi.ComponentContext, overrides OverridesInterface) (clusterapi.ApplyUpgradeOptions, error) {
	c.initializeImageVersionsOverrides(ctx.Log(), overrides)
	applyUpgradeOptions := clusterapi.ApplyUpgradeOptions{}
	podList := &v1.PodList{}
	if err := ctx.Client().List(context.TODO(), podList, &client.ListOptions{Namespace: ComponentNamespace}); err != nil {
		return applyUpgradeOptions, err
	}

	const formatString = "%s/%s:%s"
	for _, pod := range podList.Items {
		for _, co := range pod.Spec.Containers {
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetClusterAPIOverridesVersion(), overrides.GetClusterAPIBomVersion(), clusterAPIControllerImage, co.Image, c.coreProvider); ok {
				applyUpgradeOptions.CoreProvider = fmt.Sprintf(formatString, ComponentNamespace, clusterAPIProviderName, version)
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCNEBootstrapOverridesVersion(), overrides.GetOCNEBootstrapBomVersion(), clusterAPIOCNEBoostrapControllerImage, co.Image, c.bootstrapProvider); ok {
				applyUpgradeOptions.BootstrapProviders = append(applyUpgradeOptions.BootstrapProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, version))
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCNEControlPlaneOverridesVersion(), overrides.GetOCNEControlPlaneBomVersion(), clusterAPIOCNEControlPLaneControllerImage, co.Image, c.controlPlaneProvider); ok {
				applyUpgradeOptions.ControlPlaneProviders = append(applyUpgradeOptions.ControlPlaneProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, version))
			}
			if ok, version := applyUpgradeVersion(ctx.Log(), overrides.GetOCIOverridesVersion(), overrides.GetOCIBomVersion(), clusterAPIOCIControllerImage, co.Image, c.infrastructureProvider); ok {
				applyUpgradeOptions.InfrastructureProviders = append(applyUpgradeOptions.InfrastructureProviders, fmt.Sprintf(formatString, ComponentNamespace, ociProviderName, version))
			}
		}
	}

	return applyUpgradeOptions, nil
}

// isUpgradeOptionsNotEmpty returns true if any of the options is not empty
func isUpgradeOptionsNotEmpty(upgradeOptions clusterapi.ApplyUpgradeOptions) bool {
	return len(upgradeOptions.CoreProvider) != 0 ||
		len(upgradeOptions.BootstrapProviders) != 0 ||
		len(upgradeOptions.ControlPlaneProviders) != 0 ||
		len(upgradeOptions.InfrastructureProviders) != 0
}

func getComponentsToUpgrade(c clusterapi.Client, options clusterapi.ApplyUpgradeOptions) ([]client.Object, error) {
	var components []unstructured.Unstructured
	var componentObjects []client.Object
	if options.CoreProvider != "" {
		coreComponents, err := c.GetProviderComponents(clusterAPIProviderName, v1alpha3.CoreProviderType, clusterapi.ComponentsOptions{TargetNamespace: constants.VerrazzanoCAPINamespace})
		if err != nil {
			return componentObjects, err
		}
		components = append(components, coreComponents.Objs()...)
	}

	if len(options.BootstrapProviders) != 0 {
		boostrapComponents, err := c.GetProviderComponents(ocneProviderName, v1alpha3.BootstrapProviderType, clusterapi.ComponentsOptions{TargetNamespace: constants.VerrazzanoCAPINamespace})
		if err != nil {
			return componentObjects, err
		}
		components = append(components, boostrapComponents.Objs()...)
	}

	if len(options.ControlPlaneProviders) != 0 {
		controlPlaneComponents, err := c.GetProviderComponents(ocneProviderName, v1alpha3.ControlPlaneProviderType, clusterapi.ComponentsOptions{TargetNamespace: constants.VerrazzanoCAPINamespace})
		if err != nil {
			return componentObjects, err
		}
		components = append(components, controlPlaneComponents.Objs()...)
	}

	if len(options.InfrastructureProviders) != 0 {
		infrastructureComponents, err := c.GetProviderComponents(
			ociProviderName, v1alpha3.InfrastructureProviderType, clusterapi.ComponentsOptions{TargetNamespace: constants.VerrazzanoCAPINamespace})
		if err != nil {
			return componentObjects, err
		}
		components = append(components, infrastructureComponents.Objs()...)
	}
	for i := range components {
		if components[i].GetObjectKind().GroupVersionKind().Group == rbacGroup {
			componentObjects = append(componentObjects, &components[i])
		}
	}
	return componentObjects, nil
}
