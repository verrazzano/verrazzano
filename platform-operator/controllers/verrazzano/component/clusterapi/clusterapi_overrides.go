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

// createTemplateInput - create the template input for install/upgrade of the
// ClusterAPI component.
func createTemplateInput(ctx spi.ComponentContext) (*TemplateInput, error) {
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
func mergeBOMOverrides(ctx spi.ComponentContext, templateInput *TemplateInput) error {
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
func mergeUserOverrides(ctx spi.ComponentContext, templateInput *TemplateInput) error {
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
