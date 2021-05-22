// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"encoding/json"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"io/ioutil"
	"path/filepath"
)

const BomFilePath = "verrazzano-bom.json"

// Bom (bill of materials) contains product metadata for components installed by Verrazzano,
// Currently, this metadata only describes images and image repositories.
// This information is needed by the helm charts and used during install/upgrade.
type Bom struct {
	// Registry is the top level registry URI which contains the image repositories.
	// An example is ghcr.io
	Registry string `json:"registry"`

	// Components is the array of component boms in the bom
	Components []BomComponent `json:"components"`
}

// BomComponent represents a high level component, such as Istio.
// Each component has one or more sub-components.
type BomComponent struct {
	// The name of the component, for example: Istio
	Name string `json:"name"`

	// SubComponents is the array of sub-components in the component
	SubComponents []BomSubComponent `json:"subcomponents"`
}

// BomSubComponent contains the bom information for a single helm chart.
// Istio is an example of a component with several sub-component.
type BomSubComponent struct {
	Name string `json:"name"`

	// Repository is the name of the repository within a registry.  This is combined
	// with the registry value to form the image URI prefix, for example: ghcr.io/verrazzano,
	// where ghci.io is the registry and verrazzano is the repository name.
	Repository string `json:"repository"`

	// Images is the array of images for this subcomponent
	Images []BomImage `json:"images"`
}

// BomImage describes a single image used by one of the helm charts.  This structure
// is needed to render the helm chart manifest, so the image fields are correctly populated
// in the resulting YAML.  There is no helm standard which specifies the image information
// used by a template, each product usually has a custom way to do this. The helm keys fields
// in this structure specify those custom keys.
type BomImage struct {
	// ImageName specifies the name of the image tag, such as `nginx-ingress-controller`
	ImageName string `json:"image"`

	// ImageName specifies the name of the image tag, such as `0.46.0-20210510134749-abc2d2088`
	ImageTag string `json:"tag"`

	// HelmImageNameKey is the helm template key which identifies the image name.  There are a variety
	// of keys used by the different helm charts, such as `api.imageName`.  The default is `image`
	HelmImageNameKey string `json:"helmPath"`

	// HelmTagNameKey is the helm template key which identifies the image tag.  There are a variety
	// of keys used by the different helm charts, such as `api.imageVersion`.  The default is `tag`
	HelmTagNameKey string `json:"helmTagPath"`

	// HelmRegistryNameKey is the helm template key which identifies the image repository.  This is not
	// normally specified.  An example is `image.registry` in external-dns.  The default is empty string
	HelmRegistryNameKey string `json:"helmRegPath"`
}

func getImageOverrides(component string, subComponent string) (string, error) {

	path := filepath.Join(config.GetPlatformDir() + BomFilePath)
	jasonBom, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Convert the json into a to bom
	bom := Bom{}
	if err := json.Unmarshal([]byte(jasonBom), &bom); err != nil {
		return "", err
	}
	return "", err
}
