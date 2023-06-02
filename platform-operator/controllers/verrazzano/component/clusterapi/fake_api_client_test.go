// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/tree"
)

type FakeClusterAPIClient struct{}

func (f *FakeClusterAPIClient) GetProvidersConfig() ([]client.Provider, error) {
	return []client.Provider{}, nil
}

func (f *FakeClusterAPIClient) GenerateProvider(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) GetProviderComponents(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) Init(options client.InitOptions) ([]client.Components, error) {
	return []client.Components{}, nil
}

func (f *FakeClusterAPIClient) InitImages(options client.InitOptions) ([]string, error) {
	return []string{}, nil
}

func (f *FakeClusterAPIClient) GetClusterTemplate(options client.GetClusterTemplateOptions) (client.Template, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) GetKubeconfig(options client.GetKubeconfigOptions) (string, error) {
	return "", nil
}

func (f *FakeClusterAPIClient) Delete(options client.DeleteOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) Move(options client.MoveOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) Backup(options client.BackupOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) Restore(options client.RestoreOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) PlanUpgrade(options client.PlanUpgradeOptions) ([]client.UpgradePlan, error) {
	return []client.UpgradePlan{}, nil
}

func (f *FakeClusterAPIClient) PlanCertManagerUpgrade(options client.PlanUpgradeOptions) (client.CertManagerUpgradePlan, error) {
	return client.CertManagerUpgradePlan{}, nil
}

func (f *FakeClusterAPIClient) ApplyUpgrade(options client.ApplyUpgradeOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) ProcessYAML(options client.ProcessYAMLOptions) (client.YamlPrinter, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) DescribeCluster(options client.DescribeClusterOptions) (*tree.ObjectTree, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) RolloutRestart(options client.RolloutOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) RolloutPause(options client.RolloutOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) RolloutResume(options client.RolloutOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) RolloutUndo(options client.RolloutOptions) error {
	return nil
}

func (f *FakeClusterAPIClient) TopologyPlan(options client.TopologyPlanOptions) (*client.TopologyPlanOutput, error) {
	return nil, nil
}
