// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

type templateData struct {
	BootstrapNodeImage string
}

// KindBootstrapProvider is an abstraction around the KIND provider, mainly for unit test purposes
type KindBootstrapProvider interface {
	CreateCluster(config ClusterConfig) error
	DestroyCluster(config ClusterConfig) error
	GetKubeconfig(config ClusterConfig) (string, error)
}

// SetKindBootstrapProvider for unit testing, override the KIND provider
func SetKindBootstrapProvider(p KindBootstrapProvider) {
	defaultKindBootstrapProviderImpl = p
}

// ResetKindBootstrapProvider for unit testing, reset the KIND provider
func ResetKindBootstrapProvider() {
	defaultKindBootstrapProviderImpl = &kindBootstrapProviderImpl{}
}

// SetKindBootstrapProvider for unit testing, override the KIND provider
func SetCNEBootstrapProvider(p KindBootstrapProvider) {
	defaultCNEBootstrapProviderImpl = p
}

// ResetKindBootstrapProvider for unit testing, reset the KIND provider
func ResetCNEBootstrapProvider() {
	defaultCNEBootstrapProviderImpl = &cneBootstrapProviderImpl{}
}

var (
	defaultKindBootstrapProviderImpl KindBootstrapProvider = &kindBootstrapProviderImpl{}
	defaultCNEBootstrapProviderImpl  KindBootstrapProvider = &cneBootstrapProviderImpl{}
)
