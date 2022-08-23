// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"testing"
)

func TestCreateBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	bootstrapCluster := NewBoostrapCluster()
	err := bootstrapCluster.Create()
	asserts.NoError(err)
}

func TestInitBoostrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	bootstrapCluster := NewBoostrapCluster()
	asserts.NoError(bootstrapCluster.Init())
}

func TestDeleteBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	asserts.NoError(NewBoostrapCluster().Destroy())
}
