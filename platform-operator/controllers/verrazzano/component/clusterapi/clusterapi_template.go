// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import "fmt"

type TemplateInterface interface {
	GetGlobalRegistry() string
	GetClusterAPIRepository() string
	GetClusterAPITag() string
	GetClusterAPIURL() string
	GetOCIRepository() string
	GetOCITag() string
	GetOCNEBootstrapRepository() string
	GetOCNEBootstrapTag() string
	GetOCNEControlPlaneRepository() string
	GetOCNEControlPlaneTag() string
}

type TemplateInput struct {
	APIVersion                 string
	APIRepository              string
	APITag                     string
	OCIVersion                 string
	OCIRepository              string
	OCITag                     string
	OCNEBootstrapVersion       string
	OCNEBootstrapRepository    string
	OCNEBootstrapTag           string
	OCNEControlPlaneVersion    string
	OCNEControlPlaneRepository string
	OCNEControlPlaneTag        string

	Overrides *capiOverrides
}

func newTemplateContext(templateInput *TemplateInput) TemplateInterface {
	return templateInput
}

func (c TemplateInput) GetGlobalRegistry() string {
	return c.Overrides.Global.Registry
}

func (c TemplateInput) GetClusterAPIRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.Core)
}

func (c TemplateInput) GetClusterAPITag() string {
	return c.Overrides.DefaultProviders.Core.Image.Tag
}

func (c TemplateInput) GetClusterAPIURL() string {
	core := c.Overrides.DefaultProviders.Core
	if len(core.Url) > 0 {
		return ""
	}
	if len(core.Version) > 0 {
		return ""
	}
	// Return default value
	return fmt.Sprintf("/verrazzano/capi/cluster-api/%s/core-components.yaml", core.Image.BomVersion)
}

func (c TemplateInput) GetOCIRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCI)
}

func (c TemplateInput) GetOCITag() string {
	return c.Overrides.DefaultProviders.OCI.Image.Tag
}

func (c TemplateInput) GetOCNEBootstrapRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c TemplateInput) GetOCNEBootstrapTag() string {
	return c.Overrides.DefaultProviders.OCNEBootstrap.Image.Tag
}

func (c TemplateInput) GetOCNEControlPlaneRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEControlPlane)
}

func (c TemplateInput) GetOCNEControlPlaneTag() string {
	return c.Overrides.DefaultProviders.OCNEControlPlane.Image.Tag
}

func getRepositoryForProvider(template TemplateInput, provider capiProvider) string {
	return fmt.Sprintf("%s/%s", getRegistryForProvider(template, provider), provider.Image.Repository)
}

func getRegistryForProvider(template TemplateInput, provider capiProvider) string {
	registry := provider.Image.Registry
	if len(registry) == 0 {
		registry = template.Overrides.Global.Registry
	}
	return registry
}
