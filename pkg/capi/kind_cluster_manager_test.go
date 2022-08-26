// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKindClusterManager_Create(t *testing.T) {
	asserts := assert.New(t)

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Create())
}

func TestKindClusterManager_Destroy(t *testing.T) {
	asserts := assert.New(t)

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Destroy())
}

func TestKindClusterManager_Init(t *testing.T) {
	asserts := assert.New(t)
	SetCAPIInitFunc(fakeCAPINew)
	defer ResetCAPIInitFunc()

	kcm := kindClusterManager{
		config:            testBootstrapCfg,
		bootstrapProvider: &TestBootstrapProvider{},
	}

	asserts.NoError(kcm.Init())
}
