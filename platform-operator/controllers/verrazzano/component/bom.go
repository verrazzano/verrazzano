// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const defaultBomFilename = "verrazzano-bom.json"
const defaultImageKey = "image"
const slash = "/"
const tagSep = ":"

// This is the BOM file path needed for unit tests
var unitTestBomFilePath string

// SetUnitTestBomFilePath is used for unit tests
func SetUnitTestBomFilePath(p string) {
	unitTestBomFilePath = p
}

// Bom contains information related to the bill of materials along with structures to process it.
// The bom file is verrazzano-bom.json and it mainly has image information.
type Bom struct {
	// bomDoc contains the contents of the JSON bom, in go structure format.
	bomDoc BomDoc

	// subComponentMap is a map of subcomponents keyed by subcomponent name.
	subComponentMap map[string]*BomSubComponent
}

// BomDoc contains product metadata for components installed by Verrazzano.
// Currently, this metadata only describes images and image repositories.
// This information is needed by the helm charts and used during install/upgrade.
type BomDoc struct {
	// Registry is the top level registry URI which contains the image repositories.
	// An example is ghcr.io
	Registry string `json:"registry"`

	// Components is the array of component boms
	Components []BomComponent `json:"components"`
}

// BomComponent represents a high level component, such as Istio.
// Each component has one or more subcomponents.
type BomComponent struct {
	// The name of the component, for example: Istio
	Name string `json:"name"`

	// SubComponents is the array of subcomponents in the component
	SubComponents []BomSubComponent `json:"subcomponents"`
}

// BomSubComponent contains the bom information for a single helm chart.
// Istio is an example of a component with several subcomponents.
type BomSubComponent struct {
	Name string `json:"name"`

	// Repository is the name of the repository within a registry.  This is combined
	// with the registry value to form the image URL prefix, for example: ghcr.io/verrazzano,
	// where ghci.io is the registry and verrazzano is the repository name.
	Repository string `json:"repository"`

	// Images is the array of images for this subcomponent
	Images []BomImage `json:"images"`
}

// BomImage describes a single image used by one of the helm charts.  This structure
// is needed to render the helm chart manifest, so the image fields are correctly populated
// in the resulting YAML.  There is no helm standard which specifies the keys for
// image information used by a template, each product usually has a custom way to do this.
// The helm keys fields in this structure specify those custom keys.
type BomImage struct {
	// ImageName specifies the name of the image tag, such as `nginx-ingress-controller`
	ImageName string `json:"image"`

	// ImageTag specifies the name of the image tag, such as `0.46.0-20210510134749-abc2d2088`
	ImageTag string `json:"tag"`

	// HelmRegistryKey is the helm template key which identifies the image registry.  This is not
	// normally specified.  An example is `image.registry` in external-dns.  The default is empty string
	HelmRegistryKey string `json:"helmRegKey"`

	// HelmRepoKey is the helm template key which identifies the image repository.
	HelmRepoKey string `json:"helmRepoKey"`

	// HelmImageKey is the helm template key which identifies the image name.  There are a variety
	// of keys used by the different helm charts, such as `api.imageName`.  The default is `image`
	HelmImageKey string `json:"helmImageKey"`

	// HelmTagKey is the helm template key which identifies the image tag.  There are a variety
	// of keys used by the different helm charts, such as `api.imageVersion`.
	HelmTagKey string `json:"helmTagKey"`

	// HelmFullImageKey is the helm path key which identifies the image name.  There are a variety
	// of keys used by the different helm charts, such as `api.imageName`.
	HelmFullImageKey string `json:"helmFullImageKey"`
}

// keyVal defines the key, value pair used to override a single helm value
type keyValue struct {
	key   string
	value string
}

// DefaultBomFilePath returns the default path of the bom file
func DefaultBomFilePath() string {
	if unitTestBomFilePath != "" {
		return unitTestBomFilePath
	}
	return filepath.Join(config.GetPlatformDir(), defaultBomFilename)
}

// Create a new bom from a JSON file
func NewBom(bomPath string) (Bom, error) {
	jsonBom, err := ioutil.ReadFile(bomPath)
	if err != nil {
		return Bom{}, err
	}
	bom := Bom{
		subComponentMap: make(map[string]*BomSubComponent),
	}
	err = bom.init(string(jsonBom))
	if err != nil {
		return Bom{}, err
	}
	return bom, nil
}

// Initialize the BomInfo.  Load the Bom from the JSON file and build
// a map of subcomponents
func (b *Bom) init(jsonBom string) error {
	// Convert the json into a to bom
	if err := json.Unmarshal([]byte(jsonBom), &b.bomDoc); err != nil {
		return err
	}

	// Build a map of subcomponents
	for _, comp := range b.bomDoc.Components {
		for i, sub := range comp.SubComponents {
			b.subComponentMap[sub.Name] = &comp.SubComponents[i]
		}
	}
	return nil
}

// GetSubcomponent gets the bom subcomponent
func (b *Bom) GetSubcomponent(subComponentName string) (*BomSubComponent, error) {
	sc, ok := b.subComponentMap[subComponentName]
	if !ok {
		return nil, errors.New("unknown subcomponent " + subComponentName)
	}
	return sc, nil
}

// GetSubcomponentImages the imageBoms for a subcomponent
func (b *Bom) GetSubcomponentImages(subComponentName string) ([]BomImage, error) {
	sc, err := b.GetSubcomponent(subComponentName)
	if err != nil {
		return nil, err
	}
	return sc.Images, nil
}

// GetSubcomponentImageCount returns the number of subcomponent images
func (b *Bom) GetSubcomponentImageCount(subComponentName string) int {
	imageBom, ok := b.subComponentMap[subComponentName]
	if !ok {
		return 0
	}
	return len(imageBom.Images)
}

// buildImageOverrides builds the image overrides array for the subComponent.
// Each override has an array of 1-n Helm key:value pairs
func (b *Bom) buildImageOverrides(subComponentName string) ([]keyValue, error) {
	sc, ok := b.subComponentMap[subComponentName]
	if !ok {
		return nil, errors.New("unknown subcomponent " + subComponentName)
	}

	registry := b.resolveRegistry(sc)
	repo := b.resolveRepo(sc)

	// Loop through the images used by this subcomponent, building
	// the list of key:value pairs.  At the very least, this will build
	// a single value for the fully qualified image name in the format of
	// registry/repository/image.tag
	var kvs []keyValue
	for _, imageBom := range sc.Images {
		fullImageBldr := strings.Builder{}

		// Normally, the registry is the first segment of the image name, for example "ghcr.io/"
		// However, there are exceptions like in external-dns, where the registry is a separate helm field,
		// in which case the registry is omitted from the image full name.
		if imageBom.HelmRegistryKey != "" {
			kvs = append(kvs, keyValue{
				key:   imageBom.HelmRegistryKey,
				value: registry,
			})
		} else {
			fullImageBldr.WriteString(registry)
			fullImageBldr.WriteString(slash)
		}

		// Either write the repo name key value, or append it to the full image path
		if imageBom.HelmRepoKey != "" {
			kvs = append(kvs, keyValue{
				key:   imageBom.HelmRepoKey,
				value: repo,
			})
		} else {
			fullImageBldr.WriteString(repo)
			fullImageBldr.WriteString(slash)
		}

		// Either write the image name key value, or append it to the full image path
		if imageBom.HelmImageKey != "" {
			kvs = append(kvs, keyValue{
				key:   imageBom.HelmImageKey,
				value: imageBom.ImageName,
			})
		} else {
			fullImageBldr.WriteString(imageBom.ImageName)
		}

		// Either write the tag name key value, or append it to the full image path
		if imageBom.HelmTagKey != "" {
			kvs = append(kvs, keyValue{
				key:   imageBom.HelmTagKey,
				value: imageBom.ImageTag,
			})
		} else {
			fullImageBldr.WriteString(tagSep)
			fullImageBldr.WriteString(imageBom.ImageTag)
		}

		fullImagePath := fullImageBldr.String()

		// If the image path key is present the create the kv with the full image path
		if imageBom.HelmFullImageKey != "" {
			kvs = append(kvs, keyValue{
				key:   imageBom.HelmFullImageKey,
				value: fullImagePath,
			})
		}
		// Default the image key if there are no specified tags.  Keycloak theme needs this
		if len(kvs) == 0 {
			kvs = append(kvs, keyValue{
				key:   defaultImageKey,
				value: fullImagePath,
			})
		}
	}
	return kvs, nil
}

// resolveRegistry resolves the registry name using the ENV var if it exists.
func (b *Bom) resolveRegistry(sc *BomSubComponent) string {
	// Get the registry ENV override, if it doesn't exist use the default
	registry := os.Getenv(constants.RegistryOverrideEnvVar)
	if registry == "" {
		registry = b.bomDoc.Registry
	}
	return registry
}

// resolveRepo resolves the repository name using the ENV var if it exists.
func (b *Bom) resolveRepo(sc *BomSubComponent) string {
	// Get the repo ENV override.  This needs to get prepended to the bom repo
	userRepo := os.Getenv(constants.ImageRepoOverrideEnvVar)
	repo := sc.Repository
	if userRepo != "" {
		repo = userRepo + slash + repo
	}
	return repo
}
