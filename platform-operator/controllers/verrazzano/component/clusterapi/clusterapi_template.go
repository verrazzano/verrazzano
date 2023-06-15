// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
)

type TemplateInterface interface {
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

type TemplateInput struct {
	OCNEControlPlaneVersion    string
	OCNEControlPlaneRepository string
	OCNEControlPlaneTag        string

	Overrides *capiOverrides
}

func newTemplateInput() *TemplateInput {
	return &TemplateInput{}
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
	return getURLForProvider(c.Overrides.DefaultProviders.Core)
}

func (c TemplateInput) GetClusterAPIVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.Core)
}

func (c TemplateInput) GetOCIRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCI)
}

func (c TemplateInput) GetOCITag() string {
	return c.Overrides.DefaultProviders.OCI.Image.Tag
}

func (c TemplateInput) GetOCIURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCI)
}

func (c TemplateInput) GetOCIVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCI)
}

func (c TemplateInput) GetOCNEBootstrapRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c TemplateInput) GetOCNEBootstrapTag() string {
	return c.Overrides.DefaultProviders.OCNEBootstrap.Image.Tag
}

func (c TemplateInput) GetOCNEBootstrapURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c TemplateInput) GetOCNEBootstrapVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCNEBootstrap)
}

func (c TemplateInput) GetOCNEControlPlaneRepository() string {
	return getRepositoryForProvider(c, c.Overrides.DefaultProviders.OCNEControlPlane)
}

func (c TemplateInput) GetOCNEControlPlaneTag() string {
	return c.Overrides.DefaultProviders.OCNEControlPlane.Image.Tag
}

func (c TemplateInput) GetOCNEControlPlaneURL() string {
	return getURLForProvider(c.Overrides.DefaultProviders.OCNEControlPlane)
}

func (c TemplateInput) GetOCNEControlPlaneVersion() string {
	return getProviderVersion(c.Overrides.DefaultProviders.OCNEControlPlane)
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
