// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import "fmt"

type TemplateInterface interface {
	GetClusterAPIRepository() string
	GetClusterAPITag() string
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

	Global           globalOverrides
	OCNEBootstrap    capiProvider
	OCNEControlPlane capiProvider
	Core             capiProvider
	OCI              capiProvider
}

func newTemplateContext(templateInput *TemplateInput) TemplateInterface {
	return templateInput
}

func (c TemplateInput) GetClusterAPIRepository() string {
	return getRepositoryForProvider(c, c.Core)
}

func (c TemplateInput) GetClusterAPITag() string {
	return c.Core.Image.Tag
}

func (c TemplateInput) GetOCIRepository() string {
	return getRepositoryForProvider(c, c.OCI)
}

func (c TemplateInput) GetOCITag() string {
	return c.OCI.Image.Tag
}

func (c TemplateInput) GetOCNEBootstrapRepository() string {
	return getRepositoryForProvider(c, c.OCNEBootstrap)
}

func (c TemplateInput) GetOCNEBootstrapTag() string {
	return c.OCNEBootstrap.Image.Tag
}

func (c TemplateInput) GetOCNEControlPlaneRepository() string {
	return getRepositoryForProvider(c, c.OCNEControlPlane)
}

func (c TemplateInput) GetOCNEControlPlaneTag() string {
	return c.OCNEControlPlane.Image.Tag
}

func getRepositoryForProvider(template TemplateInput, provider capiProvider) string {
	return fmt.Sprintf("%s/%s", getRegistryForProvider(template, provider), provider.Image.Repository)
}

func getRegistryForProvider(template TemplateInput, provider capiProvider) string {
	registry := provider.Image.Registry
	if len(registry) == 0 {
		registry = template.Global.Registry
	}
	return registry
}
