// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

type templateData struct {
	BootstrapNodeImage string
}

// KindBootstrapProvider is an abstraction around the KIND provider, mainly for unit test purposes
type KindBootstrapProvider interface {
	CreateCluster(config ClusterConfig) error
	DestroyCluster(config ClusterConfig) error
	GetKubeconfig(config ClusterConfig) (string, error)
}

// SetKindBootstrapProvider for unit testing, override the KIND provider
func SetKindBootstrapProvider(p KindBootstrapProvider) {
	defaultKindBootstrapProviderImpl = p
}

// ResetKindBootstrapProvider for unit testing, reset the KIND provider
func ResetKindBootstrapProvider() {
	defaultKindBootstrapProviderImpl = &kindBootstrapProviderImpl{}
}

var (
	defaultKindBootstrapProviderImpl KindBootstrapProvider = &kindBootstrapProviderImpl{}
)

const defaultCNEBootstrapNodeImage = "ghcr.io/verrazzano/kind-ocne:v0.14.0-20220829221147-81d706e2"
const defaultKindBootstrapNodeImage = "kindest/node:v1.24.0"

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
const defaultKindBootstrapConfig = `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: {{.BootstrapNodeImage}}
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
`
