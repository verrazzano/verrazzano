// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/tree"
	"testing"
)

type testBootstrapProvider struct{}

func (t testBootstrapProvider) CreateCluster(_ string) error {
	return nil
}

func (t testBootstrapProvider) DestroyCluster(_ string) error {
	return nil
}

func (t testBootstrapProvider) GetKubeconfig(_ string) (string, error) {
	return "", nil
}

type fakeCAPIClient struct{}

func (f fakeCAPIClient) GetProvidersConfig() ([]client.Provider, error) {
	return []client.Provider{}, nil
}

func (f fakeCAPIClient) GetProviderComponents(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return nil, nil
}

func (f fakeCAPIClient) Init(options client.InitOptions) ([]client.Components, error) {
	return []client.Components{}, nil
}

func (f fakeCAPIClient) InitImages(options client.InitOptions) ([]string, error) {
	return []string{}, nil
}

func (f fakeCAPIClient) GetClusterTemplate(options client.GetClusterTemplateOptions) (client.Template, error) {
	return nil, nil
}

func (f fakeCAPIClient) GetKubeconfig(options client.GetKubeconfigOptions) (string, error) {
	return "", nil
}

func (f fakeCAPIClient) Delete(options client.DeleteOptions) error {
	return nil
}

func (f fakeCAPIClient) Move(options client.MoveOptions) error {
	return nil
}

func (f fakeCAPIClient) Backup(options client.BackupOptions) error {
	return nil
}

func (f fakeCAPIClient) Restore(options client.RestoreOptions) error {
	return nil
}

func (f fakeCAPIClient) PlanUpgrade(options client.PlanUpgradeOptions) ([]client.UpgradePlan, error) {
	return []client.UpgradePlan{}, nil
}

func (f fakeCAPIClient) PlanCertManagerUpgrade(options client.PlanUpgradeOptions) (client.CertManagerUpgradePlan, error) {
	return client.CertManagerUpgradePlan{}, nil
}

func (f fakeCAPIClient) ApplyUpgrade(options client.ApplyUpgradeOptions) error {
	return nil
}

func (f fakeCAPIClient) ProcessYAML(options client.ProcessYAMLOptions) (client.YamlPrinter, error) {
	return nil, nil
}

func (f fakeCAPIClient) DescribeCluster(options client.DescribeClusterOptions) (*tree.ObjectTree, error) {
	return nil, nil
}

func (f fakeCAPIClient) RolloutRestart(options client.RolloutOptions) error {
	return nil
}

func (f fakeCAPIClient) RolloutPause(options client.RolloutOptions) error {
	return nil
}

func (f fakeCAPIClient) RolloutResume(options client.RolloutOptions) error {
	return nil
}

func (f fakeCAPIClient) RolloutUndo(options client.RolloutOptions) error {
	return nil
}

func (f fakeCAPIClient) TopologyPlan(options client.TopologyPlanOptions) (*client.TopologyPlanOutput, error) {
	return nil, nil
}

func TestCreateBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetBootstrapProvider()
	defer ResetCAPIInitFunc()

	bootstrapCluster := NewBoostrapCluster()
	err := bootstrapCluster.Create()
	asserts.NoError(err)
}

func TestInitBoostrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetBootstrapProvider()
	defer ResetCAPIInitFunc()

	bootstrapCluster := NewBoostrapCluster()
	asserts.NoError(bootstrapCluster.Init())
}

func TestDeleteBootstrapCluster(t *testing.T) {
	asserts := assert.New(t)
	SetBootstrapProvider(testBootstrapProvider{})
	SetCAPIInitFunc(func(path string, options ...client.Option) (client.Client, error) {
		return &fakeCAPIClient{}, nil
	})
	defer ResetBootstrapProvider()
	defer ResetCAPIInitFunc()

	asserts.NoError(NewBoostrapCluster().Destroy())
}
