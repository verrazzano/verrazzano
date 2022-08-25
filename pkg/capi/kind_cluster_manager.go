// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"go.uber.org/zap"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

const capiDockerProvider = "docker"

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
	return initCAPI(r)
}

func (r *kindClusterManager) Destroy() error {
	return r.bootstrapProvider.DestroyCluster(r.config)
}

func initCAPI(clcm ClusterLifeCycleManager) error {
	config, err := createKubeConfigFile(clcm)
	if err != nil {
		return err
	}
	defer os2.RemoveTempFiles(zap.S(), config.Name())
	capiclient, err := capiInitFunc("")
	if err != nil {
		return err
	}
	_, err = capiclient.Init(clusterapi.InitOptions{
		Kubeconfig: clusterapi.Kubeconfig{
			Path: config.Name(),
		},
		InfrastructureProviders: defaultCAPIProviders,
	})
	return err
}
