// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kind "sigs.k8s.io/kind/pkg/cmd"
)

var bootstrapConfig = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
`

type BootstrapProvider interface {
	CreateCluster(clusterName string) error
	DestroyCluster(clusterName string) error
	GetKubeconfig(clusterName string) (string, error)
}

func SetBootstrapProvider(p BootstrapProvider) {
	bootstrapProviderImpl = p
}

func ResetBootstrapProvider() {
	bootstrapProviderImpl = &kindBootstrapProvider{}
}

var bootstrapProviderImpl BootstrapProvider = &kindBootstrapProvider{}

type kindBootstrapProvider struct{}

func (k *kindBootstrapProvider) CreateCluster(clusterName string) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.Create(clusterName, kindcluster.CreateWithRawConfig([]byte(bootstrapConfig)))
}

func (k *kindBootstrapProvider) DestroyCluster(clusterName string) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	kubeconfig, err := k.GetKubeconfig(clusterName)
	if err != nil {
		return err
	}
	return provider.Delete(clusterName, kubeconfig)
}

func (k *kindBootstrapProvider) GetKubeconfig(clusterName string) (string, error) {
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.KubeConfig(clusterName, false)
}
