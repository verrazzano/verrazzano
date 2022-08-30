// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"fmt"

	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

type CAPIInitFuncType = func(path string, options ...clusterapi.Option) (clusterapi.Client, error)

const (
	KindClusterType = "kind"
	OCNEClusterType = "ocne"
	NoClusterType   = "noCluster"
)

var capiInitFunc = clusterapi.New
var publicSupportedClusterTypes = []string{KindClusterType, OCNEClusterType}
var allSupportedClusterTypes = append(publicSupportedClusterTypes, NoClusterType)

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
	GetClusterName() string
	GetType() string
	GetContainerImage() string
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
		config:            bootstrapClusterConfig{},
		bootstrapProvider: defaultKindBootstrapProviderImpl,
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
	switch actualConfig.GetType() {
	case KindClusterType, OCNEClusterType:
		return &kindClusterManager{
			config:            actualConfig,
			bootstrapProvider: defaultKindBootstrapProviderImpl,
		}, nil
	case NoClusterType:
		return &noClusterManager{
			config: actualConfig,
		}, nil
	default:
		return nil, unknownClusterTypeError(actualConfig.GetType())
	}
}

func setDefaults(c ClusterConfig) ClusterConfig {
	defaultConfig := bootstrapClusterConfig{}
	actualConfig := ClusterConfigInfo{
		ClusterName:    c.GetClusterName(),
		Type:           c.GetType(),
		ContainerImage: c.GetContainerImage(),
	}
	if actualConfig.GetClusterName() == "" {
		actualConfig.ClusterName = defaultConfig.GetClusterName()
	}
	if actualConfig.GetType() == "" {
		actualConfig.Type = defaultConfig.GetType()
	}
	if actualConfig.GetContainerImage() == "" {
		defaultImage := getDefaultBoostrapImage(actualConfig.GetType())
		if len(defaultImage) > 0 {
			actualConfig.ContainerImage = defaultImage
		}
	}
	return actualConfig
}

func validateConfig(config ClusterConfig) error {
	valid := false
	for _, clusterType := range allSupportedClusterTypes {
		if config.GetType() == clusterType {
			valid = true
		}
	}
	if !valid {
		return unknownClusterTypeError(config.GetType())
	}
	return nil
}

func unknownClusterTypeError(clusterType string) error {
	return fmt.Errorf("unsupported cluster type %s - supported types are %v",
		clusterType, publicSupportedClusterTypes)
}
