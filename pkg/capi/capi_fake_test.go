// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/tree"
)

// Fake impls for a KindBootstrapProvider and a CAPI client for unit testing

type TestBootstrapProvider struct{}

func (t *TestBootstrapProvider) CreateCluster(config ClusterConfig) error {
	return nil
}

func (t *TestBootstrapProvider) DestroyCluster(config ClusterConfig) error {
	return nil
}

func (t *TestBootstrapProvider) GetKubeconfig(config ClusterConfig) (string, error) {
	return "", nil
}

type FakeCAPIClient struct{}

func (f *FakeCAPIClient) GetProvidersConfig() ([]client.Provider, error) {
	return []client.Provider{}, nil
}

func (f *FakeCAPIClient) GetProviderComponents(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return nil, nil
}

func (f *FakeCAPIClient) Init(options client.InitOptions) ([]client.Components, error) {
	return []client.Components{}, nil
}

func (f *FakeCAPIClient) InitImages(options client.InitOptions) ([]string, error) {
	return []string{}, nil
}

func (f *FakeCAPIClient) GetClusterTemplate(options client.GetClusterTemplateOptions) (client.Template, error) {
	return nil, nil
}

func (f *FakeCAPIClient) GetKubeconfig(options client.GetKubeconfigOptions) (string, error) {
	return "", nil
}

func (f *FakeCAPIClient) Delete(options client.DeleteOptions) error {
	return nil
}

func (f *FakeCAPIClient) Move(options client.MoveOptions) error {
	return nil
}

func (f *FakeCAPIClient) Backup(options client.BackupOptions) error {
	return nil
}

func (f *FakeCAPIClient) Restore(options client.RestoreOptions) error {
	return nil
}

func (f *FakeCAPIClient) PlanUpgrade(options client.PlanUpgradeOptions) ([]client.UpgradePlan, error) {
	return []client.UpgradePlan{}, nil
}

func (f *FakeCAPIClient) PlanCertManagerUpgrade(options client.PlanUpgradeOptions) (client.CertManagerUpgradePlan, error) {
	return client.CertManagerUpgradePlan{}, nil
}

func (f *FakeCAPIClient) ApplyUpgrade(options client.ApplyUpgradeOptions) error {
	return nil
}

func (f *FakeCAPIClient) ProcessYAML(options client.ProcessYAMLOptions) (client.YamlPrinter, error) {
	return nil, nil
}

func (f *FakeCAPIClient) DescribeCluster(options client.DescribeClusterOptions) (*tree.ObjectTree, error) {
	return nil, nil
}

func (f *FakeCAPIClient) RolloutRestart(options client.RolloutOptions) error {
	return nil
}

func (f *FakeCAPIClient) RolloutPause(options client.RolloutOptions) error {
	return nil
}

func (f *FakeCAPIClient) RolloutResume(options client.RolloutOptions) error {
	return nil
}

func (f *FakeCAPIClient) RolloutUndo(options client.RolloutOptions) error {
	return nil
}

func (f *FakeCAPIClient) TopologyPlan(options client.TopologyPlanOptions) (*client.TopologyPlanOutput, error) {
	return nil, nil
}
