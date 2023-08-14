// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"github.com/verrazzano/verrazzano/pkg/semver"
	"os"
	"sigs.k8s.io/yaml"
)

type Catalog struct {
	Modules    []Module `json:"modules"`
	versionMap map[string]*semver.SemVersion
}

type Module struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewCatalog takes a path and returns a new Catalog object
func NewCatalog(catalogPath string) (Catalog, error) {
	yamlCatalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return Catalog{}, err
	}
	return NewCatalogfromYAML(yamlCatalog)
}

// NewCatalogfromYAML Create a new Catalog instance from a yaml payload
func NewCatalogfromYAML(yamlCatalog []byte) (Catalog, error) {
	catalog := Catalog{
		versionMap: make(map[string]*semver.SemVersion),
	}
	err := catalog.init(string(yamlCatalog))
	if err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

// Initialize the Catalog object from a YAML string
func (c *Catalog) init(yamlCatalog string) error {
	// Convert the json into a to bom
	if err := yaml.Unmarshal([]byte(yamlCatalog), &c); err != nil {
		return err
	}

	// Build a map of modules
	for _, module := range c.Modules {
		version, err := semver.NewSemVersion(module.Version)
		if err != nil {
			return err
		}
		c.versionMap[module.Name] = version
	}
	return nil
}

// GetVersion returns the version for the provided module
func (c *Catalog) GetVersion(module string) *semver.SemVersion {
	return c.versionMap[module]
}
