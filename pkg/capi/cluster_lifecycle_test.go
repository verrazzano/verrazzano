// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

var testBootstrapCfg = NewClusterConfig()

func fakeCAPINew(_ string, _ ...client.Option) (client.Client, error) {
	return &FakeCAPIClient{}, nil
}

func TestCreateDefaultBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(&TestBootstrapProvider{})
	SetCAPIInitFunc(fakeCAPINew)
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	clm, err := NewBoostrapCluster(ClusterConfig{})
	asserts.NoError(err)
	asserts.NoError(clm.Create())
}

func TestInitDefaultBoostrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(&TestBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &FakeCAPIClient{}, nil
	})
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	clm, err := NewBoostrapCluster(ClusterConfig{})
	asserts.NoError(err)
	asserts.NotNil(clm)
}

func TestDeleteDefaultBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetKindBootstrapProvider(&TestBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &FakeCAPIClient{}, nil
	})
	defer ResetKindBootstrapProvider()
	defer ResetCAPIInitFunc()

	clm, err := NewBoostrapCluster(ClusterConfig{})
	asserts.NoError(err)
	asserts.NoError(clm.Destroy())
}

// Test NewBootstrapCluster with different valid and invalid configurations
func TestCreateBootstrapClusterConfigValidations(t *testing.T) {
	tests := []struct {
		clusterName    string
		clusterType    string
		containerImage string
		errExpected    bool
		// expected values are provided if different from the above values
		expectedClusterName    string
		expectedClusterType    string
		expectedContainerImage string
	}{
		{clusterName: "some-cluster", clusterType: "sometype", containerImage: "someimage", errExpected: true},
		{clusterName: "some-cluster", clusterType: "", containerImage: "someimage", errExpected: false, expectedClusterType: testBootstrapCfg.Type},
		{clusterName: "", clusterType: "", containerImage: "", errExpected: false, expectedClusterName: testBootstrapCfg.ClusterName, expectedClusterType: testBootstrapCfg.Type, expectedContainerImage: defaultCNEBootstrapNodeImage},
		{clusterName: "some-cluster", clusterType: OCNEClusterType, containerImage: "someimage", errExpected: false, expectedClusterType: testBootstrapCfg.Type},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			asserts := assert.New(t)
			SetKindBootstrapProvider(&TestBootstrapProvider{})
			SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
				return &FakeCAPIClient{}, nil
			})
			defer ResetKindBootstrapProvider()
			defer ResetCAPIInitFunc()
			config := ClusterConfig{
				ClusterName:    tt.clusterName,
				Type:           tt.clusterType,
				ContainerImage: tt.containerImage,
			}
			cluster, err := NewBoostrapCluster(config)
			if tt.errExpected {
				asserts.Error(err, "Expected error creating bootstrap cluster with config")
				return
			}
			asserts.NoError(err, "Expected no error creating bootstrap cluster with config")

			cfg := cluster.GetConfig()
			expectedClusterName := tt.expectedClusterName
			expectedClusterType := tt.expectedClusterType
			expectedContainerImage := tt.expectedContainerImage
			if expectedClusterName == "" {
				expectedClusterName = tt.clusterName
			}
			if expectedClusterType == "" {
				expectedClusterType = tt.clusterType
			}
			if expectedContainerImage == "" {
				expectedContainerImage = tt.containerImage
			}
			asserts.Equal(expectedClusterName, cfg.ClusterName)
			asserts.Equal(expectedClusterType, cfg.Type)
			asserts.Equal(expectedContainerImage, cfg.ContainerImage)
		})
	}
}
