// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"os"

	kindcluster "github.com/verrazzano/kind/pkg/cluster"
	kind "github.com/verrazzano/kind/pkg/cmd"
)

// TODO: fill this in with real image when ready
const defaultCNEBootstrapNodeImage = "kindest/node:v1.24.0"

const defaultCNEBootstrapConfig = `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    image: {{.BootstrapNodeImage}}
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        imageRepository: container-registry.oracle.com/olcne
        kubernetesVersion: 1.23.7
        etcd:
          local:
            imageRepository: container-registry.oracle.com/olcne
            imageTag: 3.5.1
        dns:
          imageRepository: container-registry.oracle.com/olcne
          imageTag: 1.8.6
        nodeRegistration:
          criSocket: unix:///var/run/crio/crio.sock
        apiServer:
          extraArgs:
            "service-account-issuer": "kubernetes.default.svc"
            "service-account-signing-key-file": "/etc/kubernetes/pki/sa.key"
      - |
        kind: InitConfiguration
        nodeRegistration:
          criSocket: unix:///var/run/crio/crio.sock
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
            authorization-mode: "AlwaysAllow"
      - |
        kind: JoinConfiguration
        nodeRegistration:
          criSocket: unix:///var/run/crio/crio.sock
    extraMounts:
      - hostPath: /var/run/docker.sock
        containerPath: /var/run/docker.sock
`

type cneBootstrapProviderImpl struct{}

func (k *cneBootstrapProviderImpl) CreateCluster(config ClusterConfig) error {
	bootstrapConfig, err := parseKindBoostrapConfig(config, defaultCNEBootstrapConfig)
	if err != nil {
		return err
	}
	//fmt.Println(fmt.Sprintf("%s", bootstrapConfig))
	provider, err := getVZKindProvider()
	if err != nil {
		return err
	}
	return provider.Create(config.GetClusterName(), kindcluster.CreateWithRawConfig(bootstrapConfig))
}

func (k *cneBootstrapProviderImpl) DestroyCluster(config ClusterConfig) error {
	provider, err := getVZKindProvider()
	if err != nil {
		return err
	}
	kubeconfig, err := k.GetKubeconfig(nil)
	if err != nil {
		return err
	}
	kubePath, err := saveKubeconfigToFile(kubeconfig)
	if err != nil {
		return err
	}
	defer func() {
		os.Remove(kubePath)
	}()
	return provider.Delete(config.GetClusterName(), kubePath)
}

func (k *cneBootstrapProviderImpl) GetKubeconfig(config ClusterConfig) (string, error) {
	provider, err := getVZKindProvider()
	if err != nil {
		return "", err
	}
	return provider.KubeConfig(config.GetClusterName(), false)
}

func getVZKindProvider() (*kindcluster.Provider, error) {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return nil, err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider, nil
}
