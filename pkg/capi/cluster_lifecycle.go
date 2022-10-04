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

	BootstrapImageEnvVar = "VZ_BOOTSTRAP_IMAGE"
	bootstrapClusterName = "vz-capi-bootstrap"
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

type ClusterConfig struct {
	ClusterName    string
	Type           string
	ContainerImage string
	CAPIProviders  []string
}

// ClusterLifeCycleManager defines the lifecycle operations of a cluster
type ClusterLifeCycleManager interface {
	GetConfig() ClusterConfig
	GetKubeConfig() (string, error)
	Create() error
	Init() error
	Destroy() error
}

// NewClusterConfig Creates a new ClusterConfig with defaults
func NewClusterConfig() ClusterConfig {
	return ClusterConfig{
		ClusterName:    bootstrapClusterName,
		Type:           OCNEClusterType,
		ContainerImage: getDefaultBoostrapImage(OCNEClusterType),
		CAPIProviders:  defaultCAPIProviders,
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
	switch actualConfig.Type {
	case KindClusterType, OCNEClusterType:
		return newKindClusterManager(actualConfig)
	case NoClusterType:
		return newNoClusterManager(actualConfig)
	default:
		return nil, unknownClusterTypeError(actualConfig.Type)
	}
}

func setDefaults(c ClusterConfig) ClusterConfig {
	defaultConfig := NewClusterConfig()
	actualConfig := ClusterConfig{
		ClusterName:    c.ClusterName,
		Type:           c.Type,
		ContainerImage: c.ContainerImage,
	}
	if actualConfig.ClusterName == "" {
		actualConfig.ClusterName = defaultConfig.ClusterName
	}
	if actualConfig.Type == "" {
		actualConfig.Type = defaultConfig.Type
	}
	if actualConfig.ContainerImage == "" {
		defaultImage := getDefaultBoostrapImage(actualConfig.Type)
		if len(defaultImage) > 0 {
			actualConfig.ContainerImage = defaultImage
		}
	}
	return actualConfig
}

func validateConfig(config ClusterConfig) error {
	valid := false
	for _, clusterType := range allSupportedClusterTypes {
		if config.Type == clusterType {
			valid = true
		}
	}
	if !valid {
		return unknownClusterTypeError(config.Type)
	}
	return nil
}

func unknownClusterTypeError(clusterType string) error {
	return fmt.Errorf("unsupported cluster type %s - supported types are %v",
		clusterType, publicSupportedClusterTypes)
}
