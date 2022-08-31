// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestNoClusterManager - mainly for code coverage
func TestNoClusterManager(t *testing.T) {
	asserts := assert.New(t)

	cm, err := newNoClusterManager(NewClusterConfig())
	asserts.NoError(err)
	asserts.NoError(cm.Create())
	asserts.NoError(cm.Destroy())
	asserts.NoError(cm.Init())

	config := cm.GetConfig()
	asserts.NotNil(config)

	kubeConfig, err := cm.GetKubeConfig()
	asserts.NoError(err)
	asserts.NotNil(kubeConfig)
}
