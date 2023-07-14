// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"bytes"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"strings"
	"text/template"
	v1 "k8s.io/api/core/v1"
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

// clusterAPIPodMatcher matches pods with an out of date CoreProvider, BootstrapProvider, ControlPlaneProvider, or InfrastructureProvider.
type clusterAPIPodMatcher struct {
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

func (c *clusterAPIPodMatcher) ReInit() error {
	images, err := getImages(istioSubcomponent, proxyv2ImageName,
		verrazzanoSubcomponent, fluentdImageName,
		wkoSubcomponent, wkoExporterImageName)
	if err != nil {
		return err
	}
	c.istioProxyImage = images[proxyv2ImageName]
	c.fluentdImage = images[fluentdImageName]
	c.wkoExporterImage = images[wkoExporterImageName]
	return nil
}

func getImages(kvs ...string) (map[string]string, error) {
	bt, err := newBomTool()
	if err != nil {
		return nil, err
	}
	if len(kvs)%2 != 0 {
		return nil, errors.New("must have even key/value pairs")
	}
	images := map[string]string{}
	for i := 0; i < len(kvs); i += 2 {
		subComponent := kvs[i]
		imageName := kvs[i+1]
		image, err := bt.getImage(subComponent, imageName)
		if err != nil {
			return nil, err
		}
		images[imageName] = image
	}
	return images, nil
}

func newBomTool() (*bomTool, error) {
	vzbom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	return &bomTool{
		vzbom: vzbom,
	}, nil
}

// isImageOutOfDate returns true if the container image is not as expected (out of date)
func isImageOutOfDate(log vzlog.VerrazzanoLogger, imageName, actualImage, expectedImage string) ImageCheckResult {
	if strings.Contains(actualImage, imageName) {
		if 0 != strings.Compare(actualImage, expectedImage) {
			return OutOfDate
		}
		return UpToDate
	}
	return NotFound
}

// Matches when a pod has an outdated istiod/proxyv2 image, or an outdate fluentd image
func (c *clusterAPIPodMatcher) Matches(log vzlog.VerrazzanoLogger, podList *v1.PodList, workloadType, workloadName string) bool {

	for i, pod := range podList.Items {
		for _, co := range pod.Spec.Containers {
			if isImageOutOfDate(log, co.Image, c.bootstrapProvider, c.) == OutOfDate {
				return true
			}
		}
	}
	return false
}
