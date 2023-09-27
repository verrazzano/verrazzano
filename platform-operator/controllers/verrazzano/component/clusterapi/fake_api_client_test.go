// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/tree"
)

type FakeClusterAPIClient struct{}

type FakeComponentCLient struct{}

func (f *FakeClusterAPIClient) GetProvidersConfig() ([]client.Provider, error) {
	return []client.Provider{}, nil
}

func (f *FakeClusterAPIClient) GenerateProvider(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return nil, nil
}

func (f *FakeClusterAPIClient) GetProviderComponents(provider string, providerType v1alpha3.ProviderType, options client.ComponentsOptions) (client.Components, error) {
	return FakeComponentCLient{}, nil
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

func (c FakeComponentCLient) Version() string {
	return ""
}

func (c FakeComponentCLient) Variables() []string {
	return nil
}

func (c FakeComponentCLient) Images() []string {
	return nil
}

func (c FakeComponentCLient) TargetNamespace() string {
	return ""
}

func (c FakeComponentCLient) InventoryObject() v1alpha3.Provider {
	return v1alpha3.Provider{}
}

func (c FakeComponentCLient) Objs() []unstructured.Unstructured {
	cm := unstructured.Unstructured{}
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	cm.SetName("test-cm")
	cm.SetNamespace(ComponentNamespace)

	role := unstructured.Unstructured{}
	role.SetGroupVersionKind(rbac.SchemeGroupVersion.WithKind("Role"))
	role.SetName("test-role")
	role.SetNamespace(ComponentNamespace)
	return []unstructured.Unstructured{cm, role}
}

func (c FakeComponentCLient) Yaml() ([]byte, error) {
	return nil, nil
}

func (c FakeComponentCLient) Name() string {
	return ""
}

func (c FakeComponentCLient) Type() v1alpha3.ProviderType {
	return ""
}

func (c FakeComponentCLient) URL() string {
	return ""
}

func (c FakeComponentCLient) SameAs(other config.Provider) bool {
	return false
}

func (c FakeComponentCLient) ManifestLabel() string {
	return ""
}

func (c FakeComponentCLient) Less(other config.Provider) bool {
	return false
}
