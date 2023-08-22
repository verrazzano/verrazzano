// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

type (
	SubnetRole string
	Subnet     struct {
		// +patchMergeKey=role
		// +patchStrategy=merge,retainKeys
		Role SubnetRole `json:"role" patchStrategy:"merge,retainKeys" patchMergeKey:"role"`
		ID   string     `json:"id"`
	}
	CommonClusterSpec struct {
		Kubernetes      `json:"kubernetes"`
		IdentityRef     NamespacedRef    `json:"identityRef"`
		PrivateRegistry *PrivateRegistry `json:"privateRegistry,omitempty"`
		Proxy           *Proxy           `json:"proxy,omitempty"`
	}
	CommonOCI struct {
		Region          string   `json:"region"`
		Compartment     string   `json:"compartment"`
		SSHPublicKey    *string  `json:"sshPublicKey,omitempty"`
		ImageName       string   `json:"imageName"`
		CloudInitScript []string `json:"cloudInitScript,omitempty"`
	}
	Kubernetes struct {
		Version        string         `json:"version"`
		ClusterNetwork ClusterNetwork `json:"clusterNetwork"`
	}
	ClusterNetwork struct {
		PodCIDR     string `json:"podCIDR"`
		ServiceCIDR string `json:"serviceCIDR"`
	}
	OCNE struct {
		Version      string           `json:"version"`
		Dependencies OCNEDependencies `json:"dependencies"`
	}
	OCNEDependencies struct {
		Install bool `json:"install"`
	}
	NodeConfig struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name          string  `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
		Shape         *string `json:"shape,omitempty"`
		OCPUs         *int    `json:"ocpus,omitempty"`
		MemoryGbs     *int    `json:"memoryGbs,omitempty"`
		BootVolumeGbs *int    `json:"bootVolumeGbs,omitempty"`
		Replicas      *int    `json:"replicas,omitempty"`
	}
	PrivateRegistry struct {
		URL               string        `json:"url"`
		CredentialsSecret NamespacedRef `json:"credentialSecret"`
	}
	Proxy struct {
		HTTPProxy  string `json:"httpProxy"`
		HTTPSProxy string `json:"httpsProxy"`
		NoProxy    string `json:"noProxy"`
	}
	NamespacedRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	QuickCreateStatus struct {
		Phase QuickCreatePhase `json:"phase"`
	}
	QuickCreatePhase string
)

// Subnet Roles
const (
	SubnetRoleControlPlane         SubnetRole = "control-plane"
	SubnetRoleControlPlaneEndpoint SubnetRole = "control-plane-endpoint"
	SubnetRoleServiceLB            SubnetRole = "service-lb"
	SubnetRoleWorker               SubnetRole = "worker"

	QuickCreatePhaseProvisioning QuickCreatePhase = "Provisioning"
	QuickCreatePhaseComplete     QuickCreatePhase = "Complete"
)
