// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestKindClusterManagerCreate - mainly for code coverage right now
func TestKindClusterManagerCreate(t *testing.T) {
	asserts := assert.New(t)

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Create())
}

// TestKindClusterManagerDestroy - mainly for code coverage right now
func TestKindClusterManagerDestroy(t *testing.T) {
	asserts := assert.New(t)

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Destroy())
}

// TestKindClusterManagerInit - mainly for code coverage right now
func TestKindClusterManagerInit(t *testing.T) {
	asserts := assert.New(t)
	SetCAPIInitFunc(fakeCAPINew)
	defer ResetCAPIInitFunc()

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Init())
}
