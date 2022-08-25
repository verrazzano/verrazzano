// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	"os"
)

// noClusterManager ClusterLifecycleManager impl for testing - does not perform any cluster operations
type noClusterManager struct {
	config ClusterConfig
}

func (r *noClusterManager) GetKubeConfig() (string, error) {
	return "", nil
}

func (r *noClusterManager) Init() error {
	fmt.Println("Init noCluster")
	return nil
}

func (r *noClusterManager) GetConfig() ClusterConfig {
	return r.config
}

func (r *noClusterManager) Create() error {
	fmt.Printf("Creating noCluster with config %v\n", r.config)
	return nil
}

func (r *noClusterManager) Destroy() error {
	fmt.Println("Destroying noCluster")
	return nil
}

func (r *noClusterManager) createKubeConfigFile() (*os.File, error) {
	return nil, nil
}
