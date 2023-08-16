// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

type (
	CommonClusterSpec struct {
		TTLSecondsAfterFinished *int             `json:"ttlSecondsAfterFinished"`
		KubernetesVersion       string           `json:"kubernetesVersion"`
		IdentityRef             NamespacedRef    `json:"identityRef"`
		PrivateRegistry         *PrivateRegistry `json:"privateRegistry,omitempty"`
		Proxy                   *Proxy           `json:"proxy,omitempty"`
	}
	CommonOCISpec struct {
		Region          string   `json:"region"`
		Compartment     string   `json:"compartment"`
		SSHPublicKey    *string  `json:"sshPublicKey,omitempty"`
		ImageName       string   `json:"imageName"`
		CloudInitScript []string `json:"cloudInitScript,omitempty"`
	}
	NodeConfig struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name          string  `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"version"`
		Shape         *string `json:"shape,omitempty"`
		OCPUs         *int    `json:"ocpus,omitempty"`
		MemoryGbs     *int    `json:"memoryGbs,omitempty"`
		BootVolumeGbs *int    `json:"bootVolumeGbs,omitempty"`
		Replicas      *int    `json:"replicas,omitempty"`
	}
	Subnets struct {
		ControlPlane string `json:"controlPlane"`
		Worker       string `json:"workers"`
		LoadBalancer string `json:"loadBalancer"`
	}
	PrivateRegistry struct {
		Server            string        `json:"server"`
		CredentialsSecret NamespacedRef `json:"credentialsSecret"`
	}
	Proxy struct {
		Server string `json:"server"`
	}
	NamespacedRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	OCNE struct {
		Version string `json:"version"`
	}
	OCNEModule struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name string `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"version"`
		Tag  string `json:"tag"`
	}
)
