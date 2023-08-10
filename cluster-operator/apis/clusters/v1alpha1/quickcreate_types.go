// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

type (
	CommonClusterSpec struct {
		KubernetesVersion string           `json:"kubernetesVersion"`
		IdentityRef       IdentityRef      `json:"identityRef"`
		Verrazzano        *Verrazzano      `json:"verrazzano,omitempty"`
		PrivateRegistry   *PrivateRegistry `json:"privateRegistry,omitempty"`
		Proxy             *Proxy           `json:"proxy,omitempty"`
	}
	IdentityRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	Verrazzano struct {
		Install     bool          `json:"install"`
		Tag         *string       `json:"tag"`
		Version     *string       `json:"version"`
		ResourceRef NamespacedRef `json:"resourceRef"`
	}
	NodeConfig struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name              string   `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"version"`
		Shape             *string  `json:"shape,omitempty"`
		OCPUs             *int     `json:"ocpus,omitempty"`
		MemoryGbs         *int     `json:"memoryGbs,omitempty"`
		BootVolumeGbs     *int     `json:"bootVolumeGbs,omitempty"`
		Replicas          *int     `json:"replicas,omitempty"`
		CloudInitCommands []string `json:"cloudInitCommands,omitempty"`
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
)
