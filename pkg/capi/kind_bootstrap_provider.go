// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package capi

import (
	"bytes"
	"io/ioutil"
	"os"
	"text/template"

	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kind "sigs.k8s.io/kind/pkg/cmd"
)

const (
	defaultCNEBootstrapNodeImage  = "ghcr.io/verrazzano/kind-ocne:v0.14.0-20220829221147-81d706e2"
	defaultKindBootstrapNodeImage = "kindest/node:v1.24.0"

	defaultCNEBootstrapConfig = `kind: Cluster
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
	defaultKindBootstrapConfig = `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: {{.BootstrapNodeImage}}
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
`
)

type kindBootstrapProviderImpl struct{}

func (k *kindBootstrapProviderImpl) CreateCluster(config ClusterConfig) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))

	bootstrapConfig, err := parseKindBoostrapConfig(config)
	if err != nil {
		return err
	}
	//fmt.Println(fmt.Sprintf("%s", bootstrapConfig))
	return provider.Create(config.ClusterName, kindcluster.CreateWithRawConfig(bootstrapConfig))
}

func (k *kindBootstrapProviderImpl) DestroyCluster(config ClusterConfig) error {
	var po kindcluster.ProviderOption
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return err
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	kubeconfig, err := k.GetKubeconfig(config)
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
	return provider.Delete(config.ClusterName, kubePath)
}

func saveKubeconfigToFile(kubeconfigContents string) (string, error) {
	// Create a temp file that contains the kubeconfig
	tmpFile, err := ioutil.TempFile("", "kubeconfig")
	if err != nil {
		return "", err
	}
	kubeConfigBytes := bytes.Buffer{}
	kubeConfigBytes.WriteString(kubeconfigContents)
	err = ioutil.WriteFile(tmpFile.Name(), kubeConfigBytes.Bytes(), 0600)
	return tmpFile.Name(), err
}

func (k *kindBootstrapProviderImpl) GetKubeconfig(config ClusterConfig) (string, error) {
	po, err := kindcluster.DetectNodeProvider()
	if err != nil {
		return "", nil
	}
	provider := kindcluster.NewProvider(po, kindcluster.ProviderWithLogger(kind.NewLogger()))
	return provider.KubeConfig(config.ClusterName, false)
}

func parseKindBoostrapConfig(config ClusterConfig) ([]byte, error) {
	kindBoostrapConfig := getDefaultBoostrapKindConfig(config.Type)
	data := templateData{BootstrapNodeImage: config.ContainerImage}
	var b bytes.Buffer
	t, err := template.New("boostrapConfig").Parse(kindBoostrapConfig)
	if err != nil {
		return []byte{}, err
	}
	if err := t.Execute(&b, &data); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

func getDefaultBoostrapImage(clusterType string) string {
	bootstrapImageOverride, envOverrideFound := os.LookupEnv(BootstrapImageEnvVar)
	if envOverrideFound {
		return bootstrapImageOverride
	}
	switch clusterType {
	case KindClusterType:
		return defaultKindBootstrapNodeImage
	case OCNEClusterType:
		return defaultCNEBootstrapNodeImage
	default:
		return ""
	}
}

func getDefaultBoostrapKindConfig(clusterType string) string {
	switch clusterType {
	case KindClusterType:
		return defaultKindBootstrapConfig
	}
	return defaultCNEBootstrapConfig
}
