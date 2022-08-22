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

const BootstrapImageEnvVar = "VZ_BOOTSTRAP_IMAGE"

func SetBootstrapProvider(p BootstrapProvider) {
	bootstrapProviderImpl = p
}

func ResetBootstrapProvider() {
	bootstrapProviderImpl = &kindBootstrapProvider{}
}

var bootstrapProviderImpl BootstrapProvider = &kindBootstrapProvider{}

type CAPIInitFuncType = func(path string, options ...clusterapi.Option) (clusterapi.Client, error)

var capiInitFunc = clusterapi.New

func SetCAPIInitFunc(f CAPIInitFuncType) {
	capiInitFunc = f
}

func ResetCAPIInitFunc() {
	capiInitFunc = clusterapi.New
}

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
	return bootstrapProviderImpl.CreateCluster(r.config.ClusterName())
}

func (r *kindClusterManager) Destroy() error {
	return bootstrapProviderImpl.DestroyCluster(r.config.ClusterName())
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
