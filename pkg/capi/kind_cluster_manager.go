// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

const capiDockerProvider = "docker"

func newKindClusterManager(actualConfig ClusterConfig) (ClusterLifeCycleManager, error) {
	return &kindClusterManager{
		config:            actualConfig,
		bootstrapProvider: defaultKindBootstrapProviderImpl,
	}, nil
}

// compile time checking for interface implementation
var _ ClusterLifeCycleManager = &kindClusterManager{}

var defaultCAPIProviders = []string{capiDockerProvider}

var _ ClusterLifeCycleManager = &kindClusterManager{}

// kindClusterManager ClusterLifecycleManager impl for a KIND-based bootstrap cluster
type kindClusterManager struct {
	config            ClusterConfig
	bootstrapProvider KindBootstrapProvider
}

func (r *kindClusterManager) GetKubeConfig() (string, error) {
	return r.bootstrapProvider.GetKubeconfig(r.config)
}

func (r *kindClusterManager) GetConfig() ClusterConfig {
	return r.config
}

func (r *kindClusterManager) Create() error {
	return r.bootstrapProvider.CreateCluster(r.config)
}

func (r *kindClusterManager) Init() error {
	return initializeCAPI(r)
}

func (r *kindClusterManager) Destroy() error {
	return r.bootstrapProvider.DestroyCluster(r.config)
}
