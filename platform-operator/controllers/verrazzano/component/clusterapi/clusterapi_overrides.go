// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
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
	VerrazzanoAddon  capiProvider `json:"verrazzanoAddon,omitempty"`
}

type capiProvider struct {
	Image        capiImage `json:"image,omitempty"`
	Version      string    `json:"version,omitempty"`
	URL          string    `json:"url,omitempty"`
	Name         string    `json:"name,omitempty"`
	MetadataFile string    `json:"metadataFile,omitempty"`
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
	GetClusterAPIControllerFullImagePath() string
	GetClusterAPITag() string
	GetClusterAPIURL() string
	GetClusterAPIVersion() string
	GetClusterAPIOverridesVersion() string
	GetClusterAPIBomVersion() string
	GetOCIRepository() string
	GetOCIControllerFullImagePath() string
	GetOCITag() string
	GetOCIURL() string
	GetOCIVersion() string
	GetOCIOverridesVersion() string
	GetOCIBomVersion() string
	GetOCNEBootstrapRepository() string
	GetOCNEBootstrapControllerFullImagePath() string
	GetOCNEBootstrapTag() string
	GetOCNEBootstrapURL() string
	GetOCNEBootstrapVersion() string
	GetOCNEBootstrapOverridesVersion() string
	GetOCNEBootstrapBomVersion() string
	GetOCNEControlPlaneRepository() string
	GetOCNEControlPlaneControllerFullImagePath() string
	GetOCNEControlPlaneTag() string
	GetOCNEControlPlaneURL() string
	GetOCNEControlPlaneVersion() string
	GetOCNEControlPlaneOverridesVersion() string
	GetOCNEControlPlaneBomVersion() string
	GetVerrazzanoAddonRepository() string
	GetVerrazzanoAddonTag() string
	GetVerrazzanoAddonVersion() string
	GetVerrazzanoAddonURL() string
	IncludeImagesHeader() bool
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

func (c capiOverrides) GetClusterAPIOverridesVersion() string {
	return c.DefaultProviders.Core.Version
}

func (c capiOverrides) GetClusterAPIBomVersion() string {
	return c.DefaultProviders.Core.Image.BomVersion
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

func (c capiOverrides) GetOCIOverridesVersion() string {
	return c.DefaultProviders.OCI.Version
}

func (c capiOverrides) GetOCIBomVersion() string {
	return c.DefaultProviders.OCI.Image.BomVersion
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

func (c capiOverrides) GetOCNEBootstrapOverridesVersion() string {
	return c.DefaultProviders.OCNEBootstrap.Version
}

func (c capiOverrides) GetOCNEBootstrapBomVersion() string {
	return c.DefaultProviders.OCNEBootstrap.Image.BomVersion
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

func (c capiOverrides) GetOCNEControlPlaneOverridesVersion() string {
	return c.DefaultProviders.OCNEControlPlane.Version
}

func (c capiOverrides) GetOCNEControlPlaneBomVersion() string {
	return c.DefaultProviders.OCNEControlPlane.Image.BomVersion
}

func (c capiOverrides) GetVerrazzanoAddonRepository() string {
	return getRepositoryForProvider(c, c.DefaultProviders.VerrazzanoAddon)
}

func (c capiOverrides) GetVerrazzanoAddonTag() string {
	return c.DefaultProviders.VerrazzanoAddon.Image.Tag
}

func (c capiOverrides) GetVerrazzanoAddonURL() string {
	return getURLForProvider(c.DefaultProviders.VerrazzanoAddon, "addon-verrazzano")
}

func (c capiOverrides) GetVerrazzanoAddonVersion() string {
	return getProviderVersion(c.DefaultProviders.VerrazzanoAddon)
}

// IncludeImagesHeader returns true if the overrides version for any of the default providers is not specified.
// Otherwise, returns false.
func (c capiOverrides) IncludeImagesHeader() bool {
	if len(c.GetClusterAPIOverridesVersion()) == 0 || len(c.GetOCIOverridesVersion()) == 0 ||
		len(c.GetOCNEBootstrapOverridesVersion()) == 0 || len(c.GetOCNEControlPlaneOverridesVersion()) == 0 {
		return true
	}
	return false
}

func (c capiOverrides) GetClusterAPIControllerFullImagePath() string {
	return fmt.Sprintf("%s/%s:%s", c.GetClusterAPIRepository(), clusterAPIControllerImage, c.GetClusterAPITag())
}

func (c capiOverrides) GetOCIControllerFullImagePath() string {
	return fmt.Sprintf("%s/%s:%s", c.GetOCIRepository(), clusterAPIOCIControllerImage, c.GetOCITag())
}

func (c capiOverrides) GetOCNEBootstrapControllerFullImagePath() string {
	return fmt.Sprintf("%s/%s:%s", c.GetOCNEBootstrapRepository(), clusterAPIOCNEBoostrapControllerImage, c.GetOCNEBootstrapTag())
}

func (c capiOverrides) GetOCNEControlPlaneControllerFullImagePath() string {
	return fmt.Sprintf("%s/%s:%s", c.GetOCNEControlPlaneRepository(), clusterAPIOCNEControlPLaneControllerImage, c.GetOCNEControlPlaneTag())
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
		return formatProviderURL(true, provider.Image.Repository, ownerRepo, provider.Version, provider.MetadataFile)
	}
	// Return default value
	return formatProviderURL(false, provider.Image.Repository, provider.Name, provider.Image.BomVersion, provider.MetadataFile)
}

// formatProviderURL - return the provider URL using the following format
// https://github.com/{owner}/{Repository}/releases/{version-tag}/{componentsClient.yaml}
func formatProviderURL(remote bool, owner string, repo string, version string, metadataFile string) string {
	if remote {
		return fmt.Sprintf("https://github.com/%s/%s/releases/%s/%s", owner, repo, version, metadataFile)
	}

	var capiRoot = "/verrazzano/capi"
	if _, err := os.Stat(capiRoot); err != nil {
		capiRoot = path.Join(config.GetPlatformDir(), "capi")
	}
	return fmt.Sprintf("%s/%s/%s/%s", capiRoot, repo, version, metadataFile)
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
	overrides.DefaultProviders.Core.Name = clusterAPIProvider
	overrides.DefaultProviders.Core.MetadataFile = "core-components.yaml"
	overrides.DefaultProviders.OCI.Name = infrastructureOciProvider
	overrides.DefaultProviders.OCI.MetadataFile = "infrastructure-components.yaml"
	overrides.DefaultProviders.OCNEBootstrap.Name = bootstrapOcneProvider
	overrides.DefaultProviders.OCNEBootstrap.MetadataFile = "bootstrap-components.yaml"
	overrides.DefaultProviders.OCNEControlPlane.Name = controlPlaneOcneProvider
	overrides.DefaultProviders.OCNEControlPlane.MetadataFile = "control-plane-components.yaml"
	overrides.DefaultProviders.VerrazzanoAddon.Name = verrazzanoAddonProvider
	overrides.DefaultProviders.VerrazzanoAddon.MetadataFile = "addon-components.yaml"
	return overrides, err
}

// mergeBOMOverrides - merge settings from the BOM
func mergeBOMOverrides(ctx spi.ComponentContext, overrides *capiOverrides) error {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return fmt.Errorf("Failed to get the BOM file for the capi image overrides: %v", err)
	}

	// Populate global values
	overrides.Global.Registry = os.Getenv(constants.RegistryOverrideEnvVar)
	if len(overrides.Global.Registry) == 0 {
		overrides.Global.Registry = bomFile.GetRegistry()
	}

	// Populate core provider values
	core := &overrides.DefaultProviders.Core.Image
	imageConfig, err := getImageOverride(ctx, bomFile, "capi-cluster-api", "capi-cluster-api", "cluster-api-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, core)

	// Populate OCI provider values
	oci := &overrides.DefaultProviders.OCI.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-oci", "capi-oci", "cluster-api-oci-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, oci)

	// Populate bootstrap provider values
	bootstrap := &overrides.DefaultProviders.OCNEBootstrap.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, bootstrap)

	// Populate controlPlane provider values
	controlPlane := &overrides.DefaultProviders.OCNEControlPlane.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, controlPlane)

	// Populate verrazzanoAddon provider values
	addon := &overrides.DefaultProviders.VerrazzanoAddon.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-addon", "capi-addon", "cluster-api-verrazzano-addon-controller")
	if err != nil {
		return err
	}
	updateImage(imageConfig, addon)

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
func getImageOverride(ctx spi.ComponentContext, bomFile bom.Bom, component string, subcomponent string, imageName string) (image *ImageConfig, err error) {
	version, err := bomFile.GetComponentVersion(component)
	if err != nil {
		return nil, err
	}

	subComp, err := bomFile.GetSubcomponent(subcomponent)
	if err != nil {
		return nil, err
	}

	img, err := bomFile.FindImage(subcomponent, imageName)
	if err != nil {
		return nil, err
	}

	repository := bomFile.ResolveRepo(subComp, img)
	if len(repository) == 0 || len(img.ImageTag) == 0 {
		return nil, ctx.Log().ErrorNewErr("Failed to find image override for %s/%s", component, imageName)
	}
	return &ImageConfig{Version: version, Repository: repository, Tag: img.ImageTag}, nil
}
