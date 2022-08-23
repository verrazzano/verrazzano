// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"os"
)

const (
	BootstrapImageEnvVar = "VZ_BOOTSTRAP_IMAGE"
	KindClusterType      = "kind"
	bootstrapClusterName = "vz-capi-bootstrap"
)

type bootstrapClusterConfig struct{}

func (r bootstrapClusterConfig) ClusterName() string {
	return bootstrapClusterName
}

func (r bootstrapClusterConfig) Type() string {
	return KindClusterType
}

func (r bootstrapClusterConfig) ContainerImage() string {
	return os.Getenv(BootstrapImageEnvVar)
}
