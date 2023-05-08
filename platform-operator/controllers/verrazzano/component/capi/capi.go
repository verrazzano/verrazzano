// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"bytes"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"strings"
	"text/template"
)

const clusterctlYamlTemplate = `
images:
  cluster-api:
    repository: {{.APIRepository}}
    tag: {{.APITag}}

  infrastructure-oci:
    repository: {{.OCIRepository}}
    tag: {{.OCITag}}

  bootstrap-ocne:
    repository: {{.OCNEBootstrapRepository}}
    tag: {{.OCNEBootstrapTag}}
  control-plane-ocne:
    repository: {{.OCNEControlPlaneRepository}}
    tag: {{.OCNEControlPlaneTag}}

providers:
  - name: "cluster-api"
    url: "/verrazzano/capi/cluster-api/{{.APIVersion}}/core-components.yaml"
    type: "CoreProvider"
  - name: "oci"
    url: "/verrazzano/capi/infrastructure-oci/{{.OCIVersion}}/infrastructure-components.yaml"
    type: "InfrastructureProvider"
  - name: "ocne"
    url: "/verrazzano/capi/bootstrap-ocne/{{.OCNEBootstrapVersion}}/bootstrap-components.yaml"
    type: "BootstrapProvider"
  - name: "ocne"
    url: "/verrazzano/capi/control-plane-ocne/{{.OCNEControlPlaneVersion}}/control-plane-components.yaml"
    type: "ControlPlaneProvider"
`

const (
	clusterTopology         = "CLUSTER_TOPOLOGY"
	expClusterResourceSet   = "EXP_CLUSTER_RESOURCE_SET"
	expMachinePool          = "EXP_MACHINE_POOL"
	initOCIClientsOnStartup = "INIT_OCI_CLIENTS_ON_STARTUP"
)

type ImageConfig struct {
	Version    string
	Repository string
	Tag        string
}

type TemplateInput struct {
	APIVersion                 string
	APIRepository              string
	APITag                     string
	OCIVersion                 string
	OCIRepository              string
	OCITag                     string
	OCNEBootstrapVersion       string
	OCNEBootstrapRepository    string
	OCNEBootstrapTag           string
	OCNEControlPlaneVersion    string
	OCNEControlPlaneRepository string
	OCNEControlPlaneTag        string
}

const (
	defaultClusterAPIDir = "/verrazzano/.cluster-api"
)

var clusterAPIDir = defaultClusterAPIDir

// Functions needed for unit testing to set and reset .cluster-api directory
func setClusterAPIDir(dir string) {
	clusterAPIDir = dir
}
func resetClusterAPIDir() {
	clusterAPIDir = defaultClusterAPIDir
}

// preInstall implementation for the CAPI Component
func preInstall(ctx spi.ComponentContext) error {
	err := setEnvVariables()
	if err != nil {
		return err
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return createClusterctlYaml(ctx)
}

// preUpgrade implementation for the CAPI Component
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
	templateInput, err := getImageOverrides(ctx)
	if err != nil {
		return err
	}

	// Apply the image overrides and versions to generate clusterctl.yaml
	result, err := applyTemplate(clusterctlYamlTemplate, templateInput)
	if err != nil {
		return err
	}

	err = os.Mkdir(clusterAPIDir, 0755)
	if err != nil {
		return err
	}

	// Create the clusterctl.yaml used when initializing CAPI.
	return os.WriteFile(clusterAPIDir+"/clusterctl.yaml", result.Bytes(), 0600)
}

// getImageOverrides returns the CAPI provider image overrides and versions from the Verrazzano bom
func getImageOverrides(ctx spi.ComponentContext) (*TemplateInput, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, ctx.Log().ErrorNewErr("Failed to get the BOM file for the capi image overrides: ", err)
	}

	templateInput := &TemplateInput{}
	imageConfig, err := getImageOverride(ctx, bomFile, "capi-cluster-api", "")
	if err != nil {
		return nil, err
	}
	templateInput.APIVersion = imageConfig.Version
	templateInput.APIRepository = imageConfig.Repository
	templateInput.APITag = imageConfig.Tag

	imageConfig, err = getImageOverride(ctx, bomFile, "capi-oci", "")
	if err != nil {
		return nil, err
	}
	templateInput.OCIVersion = imageConfig.Version
	templateInput.OCIRepository = imageConfig.Repository
	templateInput.OCITag = imageConfig.Tag

	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return nil, err
	}
	templateInput.OCNEBootstrapVersion = imageConfig.Version
	templateInput.OCNEBootstrapRepository = imageConfig.Repository
	templateInput.OCNEBootstrapTag = imageConfig.Tag

	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return nil, err
	}
	templateInput.OCNEControlPlaneVersion = imageConfig.Version
	templateInput.OCNEControlPlaneRepository = imageConfig.Repository
	templateInput.OCNEControlPlaneTag = imageConfig.Tag

	return templateInput, nil
}

// getImageOverride returns the image override and version for a given CAPI provider.
func getImageOverride(ctx spi.ComponentContext, bomFile bom.Bom, component string, imageName string) (image *ImageConfig, err error) {
	version, err := bomFile.GetComponentVersion(component)
	if err != nil {
		return nil, err
	}

	images, err := bomFile.GetImageNameList(component)
	if err != nil {
		return nil, err
	}

	var repository string
	var tag string

	for _, image := range images {
		if len(imageName) == 0 || strings.Contains(image, imageName) {
			imageSplit := strings.Split(image, ":")
			tag = imageSplit[1]
			index := strings.LastIndex(imageSplit[0], "/")
			repository = imageSplit[0][:index]
			break
		}
	}

	if len(repository) == 0 || len(tag) == 0 {
		return nil, ctx.Log().ErrorNewErr("Failed to find image override for %s/%s", component, imageName)
	}

	return &ImageConfig{Version: version, Repository: repository, Tag: tag}, nil
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
