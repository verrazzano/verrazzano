// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	bootstrapCluster := NewBoostrapCluster()
	err := bootstrapCluster.Create()
	asserts.NoError(err)
	asserts.NoError(bootstrapCluster.Init())
}

func TestInitBoostrapCluster(t *testing.T) {
	asserts := assert.New(t)
	bootstrapCluster := NewBoostrapCluster()
	asserts.NoError(bootstrapCluster.Init())
}

func TestDeleteBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	asserts.NoError(NewBoostrapCluster().Destroy())
}
