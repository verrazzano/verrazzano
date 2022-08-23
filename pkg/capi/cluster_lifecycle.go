// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"fmt"

	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

type CAPIInitFuncType = func(path string, options ...clusterapi.Option) (clusterapi.Client, error)

var capiInitFunc = clusterapi.New

// SetCAPIInitFunc For unit testing, override the CAPI init function
func SetCAPIInitFunc(f CAPIInitFuncType) {
	capiInitFunc = f
}

// ResetCAPIInitFunc For unit testing, reset the CAPI init function to its default
func ResetCAPIInitFunc() {
	capiInitFunc = clusterapi.New
}

// ClusterConfig Defines the properties of a cluster
type ClusterConfig interface {
	ClusterName() string
	Type() string
	ContainerImage() string
}

// ClusterLifeCycleManager defines the lifecycle operations of a cluster
type ClusterLifeCycleManager interface {
	GetConfig() ClusterConfig
	GetKubeConfig() (string, error)
	Create() error
	Init() error
	Destroy() error
}

// NewDefaultBoostrapCluster Creates a new cluster manager for a local bootstrap cluster with default config
func NewDefaultBoostrapCluster() ClusterLifeCycleManager {
	return &kindClusterManager{
		config: bootstrapClusterConfig{},
	}
}

// NewBoostrapCluster Creates a new cluster manager for a local bootstrap cluster with the given
// config, applying defaults where needed
func NewBoostrapCluster(clusterConfig ClusterConfig) (ClusterLifeCycleManager, error) {
	actualConfig := setDefaults(clusterConfig)
	err := validateConfig(actualConfig)
	if err != nil {
		return nil, err
	}
	return &kindClusterManager{
		config: actualConfig,
	}, nil
}

func setDefaults(c ClusterConfig) ClusterConfig {
	defaultConfig := bootstrapClusterConfig{}
	actualConfig := ClusterConfigInfo{
		ClusterNameVal:    c.ClusterName(),
		TypeVal:           c.Type(),
		ContainerImageVal: c.ContainerImage(),
	}
	if actualConfig.ClusterName() == "" {
		actualConfig.ClusterNameVal = defaultConfig.ClusterName()
	}
	if actualConfig.Type() == "" {
		actualConfig.TypeVal = defaultConfig.Type()
	}
	if actualConfig.ContainerImage() == "" {
		actualConfig.ContainerImageVal = defaultConfig.ContainerImage()
	}
	return actualConfig
}

func validateConfig(config ClusterConfig) error {
	if config.Type() != KindClusterType {
		return fmt.Errorf("Unsupported cluster type %s - only %s is supported", config.Type(), KindClusterType)
	}
	return nil
}
