// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"os"
	"path/filepath"

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
	Url           string    `json:"url,omitempty"`
	Name          string    `json:"name,omitempty"`
	MetaddataFile string    `json:"metaddataFile,omitempty"`
}

type capiImage struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	BomVersion string `json:"bomVersion,omitempty"`
}

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

type OverridesInput struct {
	Overrides *capiOverrides
}

func newTemplateInput() *OverridesInput {
	return &OverridesInput{}
}

func newTemplateContext(templateInput *OverridesInput) OverridesInterface {
	return templateInput
}

func (c OverridesInput) GetGlobalRegistry() string {
	return c.Overrides.Global.Registry
}

func (c OverridesInput) GetClusterAPIRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.Core)
}

func (c OverridesInput) GetClusterAPITag() string {
	return c.Overrides.DefaultProviders.Core.Image.Tag
}

func (c OverridesInput) GetClusterAPIURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.Core)
}

func (c OverridesInput) GetClusterAPIVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.Core)
}

func (c OverridesInput) GetOCIRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCI)
}

func (c OverridesInput) GetOCITag() string {
	return c.Overrides.DefaultProviders.OCI.Image.Tag
}

func (c OverridesInput) GetOCIURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCI)
}

func (c OverridesInput) GetOCIVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCI)
}

func (c OverridesInput) GetOCNEBootstrapRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c OverridesInput) GetOCNEBootstrapTag() string {
	return c.Overrides.DefaultProviders.OCNEBootstrap.Image.Tag
}

func (c OverridesInput) GetOCNEBootstrapURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c OverridesInput) GetOCNEBootstrapVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c OverridesInput) GetOCNEControlPlaneRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEControlPlane)
}

func (c OverridesInput) GetOCNEControlPlaneTag() string {
	return c.Overrides.DefaultProviders.OCNEControlPlane.Image.Tag
}

func (c OverridesInput) GetOCNEControlPlaneURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCNEControlPlane)
}

func (c OverridesInput) GetOCNEControlPlaneVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCNEControlPlane)
}

func getRepositoryForProvider(template OverridesInput, provider capiProvider) string {
	return fmt.Sprintf("%s/%s", getRegistryForProvider(template, provider), provider.Image.Repository)
}

func getRegistryForProvider(template OverridesInput, provider capiProvider) string {
	registry := provider.Image.Registry
	if len(registry) == 0 {
		registry = template.Overrides.Global.Registry
	}
	return registry
}

func getProviderVersion(provider capiProvider) string {
	if len(provider.Version) > 0 {
		return provider.Version
	}
	return provider.Image.BomVersion
}

func getURLForProvider(provider capiProvider) string {
	if len(provider.Url) > 0 {
		return provider.Url
	}
	if len(provider.Version) > 0 {
		return formatProviderUrl(true, provider.Name, provider.Version, provider.MetaddataFile)
	}
	// Return default value
	return formatProviderUrl(false, provider.Name, provider.Image.BomVersion, provider.MetaddataFile)
}

func formatProviderUrl(remote bool, name string, version string, metadataFile string) string {
	prefix := ""
	if remote {
		prefix = "https://github.com"
	}
	return fmt.Sprintf("%s/verrazzano/capi/%s/%s/%s", prefix, name, version, metadataFile)
}

// createTemplateInput - create the template input for install/upgrade of the
// ClusterAPI component.
func createTemplateInput(ctx spi.ComponentContext) (*OverridesInput, error) {
	templateInput := newTemplateInput()

	// Get the base overrides
	var err error
	templateInput.Overrides, err = getBaseOverrides(ctx)
	if err != nil {
		return nil, err
	}

	// Overlay base overrides with values from the BOM
	if err = mergeBOMOverrides(ctx, templateInput); err != nil {
		return nil, err
	}

	// Merge overrides from the Verrazzano custom resource
	err = mergeUserOverrides(ctx, templateInput)

	return templateInput, nil
}

// getBaseOverrides - return the base ClusterAPI overrides
func getBaseOverrides(ctx spi.ComponentContext) (*capiOverrides, error) {
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

// mergeBOMOverrides - merge settings from the BOM template
func mergeBOMOverrides(ctx spi.ComponentContext, templateInput *OverridesInput) error {
	overrides := templateInput.Overrides

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
		image.Repository = imageConfig.RepositoryWithoutRegistry
	}
	if len(imageConfig.Version) > 0 {
		image.BomVersion = imageConfig.Version
	}
}

// mergeUserOverrides - Update the struct with overrides from the VZ custom resource
func mergeUserOverrides(ctx spi.ComponentContext, templateInput *OverridesInput) error {
	overrides := templateInput.Overrides
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
