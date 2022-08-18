// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
)

//const BootstrapImageEnvVar = "VZ_BOOTSTRAP_IMAGE"

var bootstrapConfig = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
`

type ClusterConfig interface {
	ClusterName() string
	Type() string
	//ContainerImage() string
}

type bootstrapClusterConfig struct{}

func (r bootstrapClusterConfig) ClusterName() string {
	return "vz-capi-bootstrap"
}

func (r bootstrapClusterConfig) Type() string {
	return "kind"
}

//func (r bootstrapClusterConfig) ContainerImage() string {
//	return os.Getenv(BootstrapImageEnvVar)
//}

type ClusterLifeCycleManager interface {
	GetConfig() ClusterConfig
	GetKubeConfig() (string, error)
	Create() error
	Init() error
	Destroy() error
}

type kindClusterManager struct {
	config ClusterConfig
}

func (r *kindClusterManager) GetKubeConfig() (string, error) {
	po, err := cluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := cluster.NewProvider(po, cluster.ProviderWithLogger(cmd.NewLogger()))
	kubeConfig, err := provider.KubeConfig(r.config.ClusterName(), false)
	if err != nil {
		return "", err
	}
	return kubeConfig, nil
}

func (r *kindClusterManager) Init() error {
	//TODO implement me
	return nil
}

func (r *kindClusterManager) GetConfig() ClusterConfig {
	return r.config
}

func (r *kindClusterManager) Create() error {
	var po cluster.ProviderOption
	po, err := cluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := cluster.NewProvider(po, cluster.ProviderWithLogger(cmd.NewLogger()))
	err = provider.Create(r.config.ClusterName(), cluster.CreateWithRawConfig([]byte(bootstrapConfig)))
	if err != nil {
		return err
	}
	return nil
}

func (r *kindClusterManager) Destroy() error {
	var po cluster.ProviderOption
	po, err := cluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := cluster.NewProvider(po, cluster.ProviderWithLogger(cmd.NewLogger()))
	kubeconfig, err := r.GetKubeConfig()
	if err != nil {
		return err
	}
	err = provider.Delete(r.config.ClusterName(), kubeconfig)
	if err != nil {
		return err
	}
	return nil
}

func New(config ClusterConfig) ClusterLifeCycleManager {
	return &kindClusterManager{
		config: bootstrapClusterConfig{},
	}
}

func NewBoostrapCluster() ClusterLifeCycleManager {
	return &kindClusterManager{
		config: bootstrapClusterConfig{},
	}
}
