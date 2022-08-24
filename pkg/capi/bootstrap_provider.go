// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kind "sigs.k8s.io/kind/pkg/cmd"
)

const defaultKindBootstrapConfig = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
`

// KindBootstrapProvider is an abstraction around the KIND provider, mainly for unit test purposes
type KindBootstrapProvider interface {
	CreateCluster(clusterName string) error
	DestroyCluster(clusterName string) error
	GetKubeconfig(clusterName string) (string, error)
}

// SetKindBootstrapProvider for unit testing, override the KIND provider
func SetKindBootstrapProvider(p KindBootstrapProvider) {
	bootstrapProviderImpl = p
}

// ResetKindBootstrapProvider for unit testing, reset the KIND provider
func ResetKindBootstrapProvider() {
	bootstrapProviderImpl = &kindBootstrapProviderImpl{}
}

var bootstrapProviderImpl KindBootstrapProvider = &kindBootstrapProviderImpl{}

type kindBootstrapProviderImpl struct{}

func (k *kindBootstrapProviderImpl) CreateCluster(clusterName string) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.Create(clusterName, kindcluster.CreateWithRawConfig([]byte(defaultKindBootstrapConfig)))
}

func (k *kindBootstrapProviderImpl) DestroyCluster(clusterName string) error {
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

func (k *kindBootstrapProviderImpl) GetKubeconfig(clusterName string) (string, error) {
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.KubeConfig(clusterName, false)
}
