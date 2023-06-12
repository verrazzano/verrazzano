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
	OCNE ocneProvider `json:"ocne,omitempty"`
	Core capiProvider `json:"core,omitempty"`
	OCI  capiProvider `json:"oci,omitempty"`
}

type ocneProvider struct {
	Version      string       `json:"version,omitempty"`
	Bootstrap    capiProvider `json:"bootstrap,omitempty"`
	ControlPlane capiProvider `json:"controlPlane,omitempty"`
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

// getCapiOverrides - return the ClusterAPI overrides
func getCapiOverrides(ctx spi.ComponentContext) (*capiOverrides, error) {
	overrides := &capiOverrides{}
	templateInput := &TemplateInput{}

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

	// Update the struct with overrides from the BOM
	err = updateWithBOMOverrides(ctx, overrides, templateInput)

	// Merge overrides from the Verrazzano custom resource
	err = updateWithVZOverrides(ctx, overrides)

	return overrides, err
}

// updateWithBOMOverrides - update the struct with overrides from the BOM
func updateWithBOMOverrides(ctx spi.ComponentContext, overrides *capiOverrides, templateInput *TemplateInput) error {

	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return fmt.Errorf("Failed to get the BOM file for the capi image overrides: %v", err)
	}

	// Populate global values
	overrides.Global.Registry = bomFile.GetRegistry()

	// Populate core provider values
	imageConfig, err := getImageOverride(ctx, bomFile, "capi-cluster-api", "")
	if err != nil {
		return err
	}
	core := &overrides.DefaultProviders.Core
	core.Image.Repository = imageConfig.RepositoryWithoutRegistry
	core.Image.Tag = imageConfig.Tag
	templateInput.APIVersion = imageConfig.Version

	// Populate OCI provider values
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-oci", "")
	if err != nil {
		return err
	}
	oci := &overrides.DefaultProviders.OCI
	oci.Image.Repository = imageConfig.RepositoryWithoutRegistry
	oci.Image.Tag = imageConfig.Tag
	templateInput.OCIVersion = imageConfig.Version

	// Populate bootstrap provider values
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-bootstrap-controller")
	if err != nil {
		return err
	}
	bootstrap := &overrides.DefaultProviders.OCNE.Bootstrap
	bootstrap.Image.Repository = imageConfig.RepositoryWithoutRegistry
	bootstrap.Image.Tag = imageConfig.Tag
	templateInput.OCNEBootstrapVersion = imageConfig.Version

	// Populate controlPlane provider values
	imageConfig, err = getImageOverride(ctx, bomFile, "capi-ocne", "cluster-api-ocne-control-plane-controller")
	if err != nil {
		return err
	}
	controlPlane := &overrides.DefaultProviders.OCNE.ControlPlane
	controlPlane.Image.Repository = imageConfig.RepositoryWithoutRegistry
	controlPlane.Image.Tag = imageConfig.Tag
	templateInput.OCNEControlPlaneVersion = imageConfig.Version

	return nil
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
