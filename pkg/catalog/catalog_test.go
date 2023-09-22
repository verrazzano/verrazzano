// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"testing"
)

const catalogPath = "../../platform-operator/manifests/catalog/catalog.yaml"
const bomPath = "../../platform-operator/verrazzano-bom.json"

type bomSubcomponentOverrides struct {
	subcomponentName string
	imageName        string
}

var modulesNotInBom = []string{
	"verrazzano-grafana-dashboards",
	"verrazzano-network-policies",
	"cluster-issuer",
}

var subcomponentOverrides = map[string]bomSubcomponentOverrides{
	"cluster-api":                 {subcomponentName: "capi-cluster-api", imageName: "cluster-api-controller"},
	"fluentbit-opensearch-output": {subcomponentName: "fluent-operator", imageName: "fluent-bit"},
	"opensearch":                  {subcomponentName: "verrazzano-monitoring-operator", imageName: "opensearch"},
	"opensearch-dashboards":       {subcomponentName: "verrazzano-monitoring-operator", imageName: "opensearch-dashboards"},
	"grafana":                     {subcomponentName: "verrazzano-monitoring-operator", imageName: "grafana"},
}

// TestCatalogModuleVersions makes sure the module versions in the catalog are up-to-date with the Bom
func TestNewCatalogModuleVersions(t *testing.T) {
	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)

	vzBOM, err := bom.NewBom(bomPath)
	assert.NoError(t, err)
	assert.NotNil(t, vzBOM)

	var bomVersion string
	for _, module := range catalog.Modules {
		if vzString.SliceContainsString(modulesNotInBom, module.Name) {
			continue
		}
		subcomponent, ok := subcomponentOverrides[module.Name]
		if ok {
			bomSubcomponent, err := vzBOM.GetSubcomponent(subcomponent.subcomponentName)
			assert.NoError(t, err)
			image, err := vzBOM.FindImage(bomSubcomponent, subcomponent.imageName)
			assert.NoError(t, err)
			imageTagSemver, err := semver.NewSemVersion(image.ImageTag)
			assert.NoError(t, err)
			assert.NotNil(t, imageTagSemver)
			bomVersion = imageTagSemver.ToStringWithoutBuildAndPrerelease()
		} else {
			bomVersion, err = vzBOM.GetComponentVersion(module.Name)
			assert.NoError(t, err)
		}
		assert.Equalf(t, bomVersion, module.Version,
			"Catalog entry for module %s out of date, BOM version: %s, catalog version %s", module.Name, bomVersion, module.Version)
	}
}

func TestGetVersionForAllRegistryComponents(t *testing.T) {
	catalog, err := NewCatalog(catalogPath)
	assert.NoError(t, err)
	assert.NotNil(t, catalog)

	for _, component := range registry.GetComponents() {
		assert.NotNilf(t, catalog.GetVersion(component.Name()), "Failed to find matching catalog version for components %s", component.Name())
	}
}
