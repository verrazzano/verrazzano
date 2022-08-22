// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kind "sigs.k8s.io/kind/pkg/cmd"
)

const BootstrapImageEnvVar = "VZ_BOOTSTRAP_IMAGE"

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
	ContainerImage() string
}

type bootstrapClusterConfig struct{}

func (r bootstrapClusterConfig) ClusterName() string {
	return "vz-capi-bootstrap"
}

func (r bootstrapClusterConfig) Type() string {
	return "kind"
}

func (r bootstrapClusterConfig) ContainerImage() string {
	return os.Getenv(BootstrapImageEnvVar)
}

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
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	kubeConfig, err := provider.KubeConfig(r.config.ClusterName(), false)
	if err != nil {
		return "", err
	}
	return kubeConfig, nil
}

func (r *kindClusterManager) Init() error {
	config, err := r.createKubeConfigFile()
	if err != nil {
		return err
	}
	defer os2.RemoveTempFiles(zap.S(), config.Name())
	capiclient, err := clusterapi.New("") // TODO: do we need to provide a CAPI config?
	if err != nil {
		return err
	}
	_, err = capiclient.Init(clusterapi.InitOptions{
		Kubeconfig: clusterapi.Kubeconfig{
			Path: config.Name(),
		},
		InfrastructureProviders: []string{"docker"},
	})
	return err
}

func (r *kindClusterManager) createKubeConfigFile() (*os.File, error) {
	kcFile, err := ioutil.TempFile(os.TempDir(), "kubeconfig-"+r.config.ClusterName())
	if err != nil {
		return nil, err
	}
	config, err := r.GetKubeConfig()
	if _, err := kcFile.Write([]byte(config)); err != nil {
		return nil, err
	}
	return kcFile, nil
}

func (r *kindClusterManager) GetConfig() ClusterConfig {
	return r.config
}

func (r *kindClusterManager) Create() error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	err = provider.Create(r.config.ClusterName(), kindcluster.CreateWithRawConfig([]byte(bootstrapConfig)))
	if err != nil {
		return err
	}
	return nil
}

func (r *kindClusterManager) Destroy() error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
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
