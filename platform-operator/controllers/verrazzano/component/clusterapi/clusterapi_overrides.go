// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/yaml"
)

// Structs to unmarshall the cluster-api-values.yaml
type capiOverrides struct {
	Global           globalOverrides  `json:"global,omitempty"`
	DefaultProviders defaultProviders `json:"defaultProviders,omitempty"`
}

type globalOverrides struct {
	Registry string `json:"registry,omitempty"`
}

type defaultProviders struct {
	OCNEBootstrap    capiProvider `json:"ocneBootstrap,omitempty"`
	OCNEControlPlane capiProvider `json:"ocneControlPlane,omitempty"`
	Core             capiProvider `json:"core,omitempty"`
	OCI              capiProvider `json:"oci,omitempty"`
}

type capiProvider struct {
	Image         capiImage `json:"image,omitempty"`
	Version       string    `json:"version,omitempty"`
	URL           string    `json:"url,omitempty"`
	Name          string    `json:"name,omitempty"`
	MetaddataFile string    `json:"metaddataFile,omitempty"`
}

type capiImage struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	BomVersion string `json:"bomVersion,omitempty"`
}

// OverridesInterface - interface for retrieving values from the Cluster API overrides
type OverridesInterface interface {
	GetGlobalRegistry() string
	GetClusterAPIRepository() string
	GetClusterAPITag() string
	GetClusterAPIURL() string
	GetClusterAPIVersion() string
	GetOCIRepository() string
	GetOCITag() string
	GetOCIURL() string
	GetOCIVersion() string
	GetOCNEBootstrapRepository() string
	GetOCNEBootstrapTag() string
	GetOCNEBootstrapURL() string
	GetOCNEBootstrapVersion() string
	GetOCNEControlPlaneRepository() string
	GetOCNEControlPlaneTag() string
	GetOCNEControlPlaneURL() string
	GetOCNEControlPlaneVersion() string
}

func newOverridesContext(overrides *capiOverrides) OverridesInterface {
	return overrides
}

func (c capiOverrides) GetGlobalRegistry() string {
	return c.Global.Registry
}

func (c capiOverrides) GetClusterAPIRepository() string {
	return getRepositoryForProvider(c, c.DefaultProviders.Core)
}

func (c capiOverrides) GetClusterAPITag() string {
	return c.DefaultProviders.Core.Image.Tag
}

func (c capiOverrides) GetClusterAPIURL() string {
	return getURLForProvider(c.DefaultProviders.Core, "cluster-api")
}

func (c capiOverrides) GetClusterAPIVersion() string {
	return getProviderVersion(c.DefaultProviders.Core)
}

func (c capiOverrides) GetOCIRepository() string {
	return getRepositoryForProvider(c, c.DefaultProviders.OCI)
}

func (c capiOverrides) GetOCITag() string {
	return c.DefaultProviders.OCI.Image.Tag
}

func (c capiOverrides) GetOCIURL() string {
	return getURLForProvider(c.DefaultProviders.OCI, "cluster-api-provider-oci")
}

func (c capiOverrides) GetOCIVersion() string {
	return getProviderVersion(c.DefaultProviders.OCI)
}

func (c capiOverrides) GetOCNEBootstrapRepository() string {
	return getRepositoryForProvider(c, c.DefaultProviders.OCNEBootstrap)
}

func (c capiOverrides) GetOCNEBootstrapTag() string {
	return c.DefaultProviders.OCNEBootstrap.Image.Tag
}

func (c capiOverrides) GetOCNEBootstrapURL() string {
	return getURLForProvider(c.DefaultProviders.OCNEBootstrap, "cluster-api-provider-ocne")
}

func (c capiOverrides) GetOCNEBootstrapVersion() string {
	return getProviderVersion(c.DefaultProviders.OCNEBootstrap)
}

func (c capiOverrides) GetOCNEControlPlaneRepository() string {
	return getRepositoryForProvider(c, c.DefaultProviders.OCNEControlPlane)
}

func (c capiOverrides) GetOCNEControlPlaneTag() string {
	return c.DefaultProviders.OCNEControlPlane.Image.Tag
}

func (c capiOverrides) GetOCNEControlPlaneURL() string {
	return getURLForProvider(c.DefaultProviders.OCNEControlPlane, "cluster-api-provider-ocne")
}

func (c capiOverrides) GetOCNEControlPlaneVersion() string {
	return getProviderVersion(c.DefaultProviders.OCNEControlPlane)
}

// getRepositoryForProvider - return the repository in the format that clusterctl
// expects (registry/owner)
func getRepositoryForProvider(overrides capiOverrides, provider capiProvider) string {
	return fmt.Sprintf("%s/%s", getRegistryForProvider(overrides, provider), provider.Image.Repository)
}

// getRegistryForProvider - return the registry value.  The value returned is either the
// global setting, or the local override for the provider.
func getRegistryForProvider(overrides capiOverrides, provider capiProvider) string {
	registry := provider.Image.Registry
	if len(registry) == 0 {
		registry = overrides.Global.Registry
	}
	return registry
}

// getProviderVersion - return the version tag.  It is either the value from the BOM,
// or the local override for the provider.
func getProviderVersion(provider capiProvider) string {
	if len(provider.Version) > 0 {
		return provider.Version
	}
	return provider.Image.BomVersion
}

// getURLForProvider - return the URL for the provider.  It is either a local override of
// the full URL, a URL derived from a version, or the default is a local file path on the
// container image.
func getURLForProvider(provider capiProvider, ownerRepo string) string {
	if len(provider.URL) > 0 {
		return provider.URL
	}
	if len(provider.Version) > 0 {
		return formatProviderURL(true, provider.Image.Repository, ownerRepo, provider.Version, provider.MetaddataFile)
	}
	// Return default value
	return formatProviderURL(false, provider.Image.Repository, provider.Name, provider.Image.BomVersion, provider.MetaddataFile)
}

// formatProviderURL - return the provider URL using the following format
// https://github.com/{owner}/{Repository}/releases/{version-tag}/{componentsClient.yaml}
func formatProviderURL(remote bool, owner string, repo string, version string, metadataFile string) string {
	if remote {
		return fmt.Sprintf("https://github.com/%s/%s/releases/%s/%s", owner, repo, version, metadataFile)
	}
	return fmt.Sprintf("/verrazzano/capi/%s/%s/%s", repo, version, metadataFile)
}

// createOverrides - create the overrides input for install/upgrade of the
// ClusterAPI component.
func createOverrides(ctx spi.ComponentContext) (*capiOverrides, error) {
	// Get the base overrides
	overrides, err := getBaseOverrides()
	if err != nil {
		return nil, err
	}

	// Overlay base overrides with values from the BOM
	if err = mergeBOMOverrides(ctx, overrides); err != nil {
		return nil, err
	}

	// Merge overrides from the Verrazzano custom resource
	err = mergeUserOverrides(ctx, overrides)

	return overrides, err
}

// getBaseOverrides - return the base ClusterAPI overrides
func getBaseOverrides() (*capiOverrides, error) {
	overrides := &capiOverrides{}

	// Unmarshall the base overrides values file into a local struct
	filePath := filepath.Join(config.GetHelmOverridesDir(), "cluster-api-values.yaml")
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, overrides)
	if err != nil {
		return nil, err
	}

	// Initialize internal static values
	overrides.DefaultProviders.Core.Name = "cluster-api"
	overrides.DefaultProviders.Core.MetaddataFile = "core-components.yaml"
	overrides.DefaultProviders.OCI.Name = "infrastructure-oci"
	overrides.DefaultProviders.OCI.MetaddataFile = "infrastructure-components.yaml"
	overrides.DefaultProviders.OCNEBootstrap.Name = "bootstrap-ocne"
	overrides.DefaultProviders.OCNEBootstrap.MetaddataFile = "bootstrap-components.yaml"
	overrides.DefaultProviders.OCNEControlPlane.Name = "control-plane-ocne"
	overrides.DefaultProviders.OCNEControlPlane.MetaddataFile = "control-plane-components.yaml"

	return overrides, err
}

// mergeBOMOverrides - merge settings from the BOM
func mergeBOMOverrides(ctx spi.ComponentContext, overrides *capiOverrides) error {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return fmt.Errorf("Failed to get the BOM file for the capi image overrides: %v", err)
	}

	// Populate global values
	overrides.Global.Registry = bomFile.GetRegistry()

	// Populate core provider values
	core := &overrides.DefaultProviders.Core.Image
	imageConfig, err := getImageOverride(ctx, bomFile, "capi-cluster-api", "")
	if err != nil {
		return err
	}
	updateImage(imageConfig, core)

	// Populate OCI provider values
	oci := &overrides.DefaultProviders.OCI.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-oci", "")
	if err != nil {
		return err
	}
	updateImage(imageConfig, oci)

	// Populate bootstrap provider values
	bootstrap := &overrides.DefaultProviders.OCNEBootstrap.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, bootstrap)

	// Populate controlPlane provider values
	controlPlane := &overrides.DefaultProviders.OCNEControlPlane.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, controlPlane)

	return nil
}

func updateImage(imageConfig *ImageConfig, image *capiImage) {
	if len(imageConfig.Tag) > 0 {
		image.Tag = imageConfig.Tag
	}
	if len(imageConfig.Repository) > 0 {
		image.Repository = imageConfig.Repository
	}
	if len(imageConfig.Version) > 0 {
		image.BomVersion = imageConfig.Version
	}
}

// mergeUserOverrides - Update the struct with overrides from the VZ custom resource
func mergeUserOverrides(ctx spi.ComponentContext, overrides *capiOverrides) error {
	if ctx.EffectiveCR().Spec.Components.ClusterAPI == nil {
		return nil
	}

	// Get install overrides as array of yaml strings
	overridesYAML, err := override.GetInstallOverridesYAML(ctx, ctx.EffectiveCR().Spec.Components.ClusterAPI.ValueOverrides)
	if err != nil {
		return err
	}

	// Convert base struct to yaml
	baseYAML, err := yaml.Marshal(overrides)
	if err != nil {
		return err
	}

	// Prepend base YAML to overrides
	allYAML := append([]string{string(baseYAML)}, overridesYAML...)

	// Perform strategic merge of overrides
	merged, err := vzyaml.StrategicMerge(capiOverrides{}, allYAML...)
	if err != nil {
		return err
	}

	// Update the struct with the resulting YAML
	return yaml.Unmarshal([]byte(merged), overrides)
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

	subComp, err := bomFile.GetSubcomponent(component)
	if err != nil {
		return nil, err
	}

	var tag string

	for _, image := range images {
		if len(imageName) == 0 || strings.Contains(image, imageName) {
			imageSplit := strings.Split(image, ":")
			tag = imageSplit[1]
			break
		}
	}

	if len(subComp.Repository) == 0 || len(tag) == 0 {
		return nil, ctx.Log().ErrorNewErr("Failed to find image override for %s/%s", component, imageName)
	}

	return &ImageConfig{Version: version, Repository: subComp.Repository, Tag: tag}, nil
}
