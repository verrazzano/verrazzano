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
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// Structs to unmarshall the cluster-api-values.yaml
type capiOverrides struct {
	Global           globalOverrides  `json:"global,omitempty"`
	DefaultProviders defaultProviders `json:"defaultProviders,omitempty"`
}

type globalOverrides struct {
	Registry         string                        `json:"registry,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	PullPolicy       corev1.PullPolicy             `json:"pullPolicy,omitempty"`
}

type defaultProviders struct {
	OCNE capiProvider `json:"ocne,omitempty"`
	Core capiProvider `json:"core,omitempty"`
	OCI  capiProvider `json:"oci,omitempty"`
}

type capiProvider struct {
	Image capiImage `json:"image,omitempty"`
}

type capiImage struct {
	Registry   string            `json:"registry,omitempty"`
	Repository string            `json:"repository,omitempty"`
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
	Tag        string            `json:"tag,omitempty"`
}

// createTemplateInput - create the template input for install/upgrade of the
// ClusterAPI component.
func createTemplateInput(ctx spi.ComponentContext) (*TemplateInput, error) {
	templateInput := &TemplateInput{}

	// Get the user facing overrides
	userOverrides, err := getCapiOverrides(ctx)
	if err != nil {
		return nil, err
	}

	// Convert the user facing overrides into the internal format for the overrides.
	// This is required because some user facing overrides (e.g. tag for ocne) is simplified
	// for the user to a single input, but internally the BOM can have different tags for
	// bootstrap and controlPlane images.
	convertUserOverridesToTemplate(templateInput, userOverrides)

	// Merge the BOM settings into the template input
	if err = mergeBOMOverrides(ctx, templateInput); err != nil {
		return nil, err
	}

	return templateInput, nil
}

// getCapiOverrides - return the ClusterAPI overrides
func getCapiOverrides(ctx spi.ComponentContext) (*capiOverrides, error) {
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

	// Merge overrides from the Verrazzano custom resource
	err = updateWithVZOverrides(ctx, overrides)

	return overrides, err
}

// convertUserOverridesToTemplate - convert the user facing overrides into the internal structure format.
func convertUserOverridesToTemplate(template *TemplateInput, overrides *capiOverrides) {
	template.Global = overrides.Global
	template.OCNEBootstrap.Image = overrides.DefaultProviders.OCNE.Image
	template.OCNEControlPlane.Image = overrides.DefaultProviders.OCNE.Image
	template.OCI.Image = overrides.DefaultProviders.OCI.Image
	template.Core.Image = overrides.DefaultProviders.Core.Image
}

// mergeBOMOverrides - merge settings from the BOM template, being careful not to unset any
// values there were overridden by the user
func mergeBOMOverrides(ctx spi.ComponentContext, templateInput *TemplateInput) error {

	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return fmt.Errorf("Failed to get the BOM file for the capi image overrides: %v", err)
	}

	// Populate global values
	if templateInput.Global.Registry == "" {
		templateInput.Global.Registry = bomFile.GetRegistry()
	}

	// Populate core provider values
	core := &templateInput.Core.Image
	imageConfig, err := getImageOverride(ctx, bomFile, "capi-cluster-api", "")
	if err != nil {
		return err
	}
	mergeImage(imageConfig, core)
	templateInput.APIVersion = imageConfig.Version

	// Populate OCI provider values
	oci := &templateInput.OCI.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-oci", "")
	if err != nil {
		return err
	}
	mergeImage(imageConfig, oci)
	templateInput.OCIVersion = imageConfig.Version

	// Populate bootstrap provider values
	bootstrap := &templateInput.OCNEBootstrap.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return err
	}
	mergeImage(imageConfig, bootstrap)
	templateInput.OCNEBootstrapVersion = imageConfig.Version

	// Populate controlPlane provider values
	controlPlane := &templateInput.OCNEControlPlane.Image
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return err
	}
	mergeImage(imageConfig, controlPlane)
	templateInput.OCNEControlPlaneVersion = imageConfig.Version

	return nil
}

func mergeImage(imageConfig *ImageConfig, image *capiImage) {
	if image.Tag == "" {
		image.Tag = imageConfig.Tag
	}
	if image.Repository == "" {
		image.Repository = imageConfig.RepositoryWithoutRegistry
	}
}

// updateWithVZOverrides - Update the struct with overrides from the VZ custom resource
func updateWithVZOverrides(ctx spi.ComponentContext, overrides *capiOverrides) error {
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
