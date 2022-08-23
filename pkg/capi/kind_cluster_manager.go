// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

const capiDockerProvider = "docker"

var defaultCAPIProviders = []string{capiDockerProvider}

// kindClusterManager ClusterLifecycleManager impl for a KIND-based bootstrap cluster
type kindClusterManager struct {
	config ClusterConfig
}

func (r *kindClusterManager) GetKubeConfig() (string, error) {
	return bootstrapProviderImpl.GetKubeconfig(r.config.ClusterName())
}

func (r *kindClusterManager) Init() error {
	config, err := r.createKubeConfigFile()
	if err != nil {
		return err
	}
	defer os2.RemoveTempFiles(zap.S(), config.Name())
	capiclient, err := capiInitFunc("") // TODO: do we need to provide a CAPI config?
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

func (r *kindClusterManager) GetConfig() ClusterConfig {
	return r.config
}

func (r *kindClusterManager) Create() error {
	return bootstrapProviderImpl.CreateCluster(r.config.ClusterName())
}

func (r *kindClusterManager) Destroy() error {
	return bootstrapProviderImpl.DestroyCluster(r.config.ClusterName())
}

func (r *kindClusterManager) createKubeConfigFile() (*os.File, error) {
	kcFile, err := ioutil.TempFile(os.TempDir(), "kubeconfig-"+r.config.ClusterName())
	if err != nil {
		return nil, err
	}
	config, err := r.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	if _, err := kcFile.Write([]byte(config)); err != nil {
		return nil, err
	}
	return kcFile, nil
}
