// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"os"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/template"
)

const clusterctlYamlTemplate = `
images:
  cluster-api:
    repository: {{.GetClusterAPIRepository}}
    tag: {{.GetClusterAPITag}}

  infrastructure-oci:
    repository: {{.GetOCIRepository}}
    tag: {{.GetOCITag}}

  bootstrap-ocne:
    repository: {{.GetOCNEBootstrapRepository}}
    tag: {{.GetOCNEBootstrapTag}}

  control-plane-ocne:
    repository: {{.GetOCNEControlPlaneRepository}}
    tag: {{.GetOCNEControlPlaneTag}}

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

type ImageCheckResult int

const (
	OutOfDate ImageCheckResult = iota
	UpToDate
	NotFound
)

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

func (c *PodMatcherClusterAPI) initializeImageVersionsOverrides(log vzlog.VerrazzanoLogger, overrides OverridesInterface) error {
	c.coreProvider = overrides.GetClusterAPIControllerFullImagePath()
	c.infrastructureProvider = overrides.GetOCIControllerFullImagePath()
	c.bootstrapProvider = overrides.GetOCNEBootstrapControllerFullImagePath()
	c.controlPlaneProvider = overrides.GetOCNEControlPlaneControllerFullImagePath()
	return nil
}

// isImageOutOfDate returns true if the container image is not as expected (out of date)
func isImageOutOfDate(log vzlog.VerrazzanoLogger, imageName, actualImage, expectedImage string) ImageCheckResult {
	log.Infof("Checking if the image %v is out of date,  actual image: %v, expected image: %v", imageName, actualImage, expectedImage)
	if strings.Contains(actualImage, imageName) {
		if 0 != strings.Compare(actualImage, expectedImage) {
			return OutOfDate
		}
		return UpToDate
	}
	return NotFound
}

// MatchAndPrepareUpgradeOptions when a pod has an outdated cluster api controllers images and prepares upgrade options for outdated images.
func (c *PodMatcherClusterAPI) matchAndPrepareUpgradeOptions(ctx spi.ComponentContext, overrides OverridesInterface) (clusterapi.ApplyUpgradeOptions, error) {
	c.initializeImageVersionsOverrides(ctx.Log(), overrides)
	applyUpgradeOptions := clusterapi.ApplyUpgradeOptions{}
	podList := &v1.PodList{}
	if err := ctx.Client().List(context.TODO(), podList, &client.ListOptions{Namespace: ComponentNamespace}); err != nil {
		return applyUpgradeOptions, err
	}
	fmt.Println("Inside Prepare Options")
	const formatString = "%s/%s:%s"
	for _, pod := range podList.Items {
		for _, co := range pod.Spec.Containers {
			if isImageOutOfDate(ctx.Log(), clusterAPIControllerImage, co.Image, c.coreProvider) == OutOfDate {
				applyUpgradeOptions.CoreProvider = fmt.Sprintf(formatString, ComponentNamespace, clusterAPIProviderName, overrides.GetClusterAPIVersion())
			}
			if isImageOutOfDate(ctx.Log(), clusterAPIOCNEBoostrapControllerImage, co.Image, c.bootstrapProvider) == OutOfDate {
				applyUpgradeOptions.BootstrapProviders = append(applyUpgradeOptions.BootstrapProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, overrides.GetOCNEBootstrapVersion()))
			}
			if isImageOutOfDate(ctx.Log(), clusterAPIOCNEControlPLaneControllerImage, co.Image, c.controlPlaneProvider) == OutOfDate {
				applyUpgradeOptions.ControlPlaneProviders = append(applyUpgradeOptions.ControlPlaneProviders, fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, overrides.GetOCNEControlPlaneVersion()))
			}
			if isImageOutOfDate(ctx.Log(), clusterAPIOCIControllerImage, co.Image, c.infrastructureProvider) == OutOfDate {
				applyUpgradeOptions.InfrastructureProviders = append(applyUpgradeOptions.InfrastructureProviders, fmt.Sprintf(formatString, ComponentNamespace, ociProviderName, overrides.GetOCIVersion()))
			}
		}
	}
	return applyUpgradeOptions, nil
}

// isUpgradeOptionsEmpty returns true if any of the options is not empty
func isUpgradeOptionsEmpty(upgradeOptions clusterapi.ApplyUpgradeOptions) bool {
	return len(upgradeOptions.CoreProvider) == 0 ||
		len(upgradeOptions.BootstrapProviders) == 0 ||
		len(upgradeOptions.ControlPlaneProviders) == 0 ||
		len(upgradeOptions.InfrastructureProviders) == 0
}
