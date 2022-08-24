// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

type CAPIInitFuncType = func(path string, options ...clusterapi.Option) (clusterapi.Client, error)

var capiInitFunc = clusterapi.New

//SetCAPIInitFunc For unit testing, override the CAPI init function
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

// NewBoostrapCluster Creaets a new cluster manager for a local bootstrap cluster
func NewBoostrapCluster() ClusterLifeCycleManager {
	return &kindClusterManager{
		config: bootstrapClusterConfig{},
	}
}
