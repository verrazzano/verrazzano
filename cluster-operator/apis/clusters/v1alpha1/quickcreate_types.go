// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

type (
	SubnetRole string
	Subnet     struct {
		// +patchMergeKey=role
		// +patchStrategy=merge,retainKeys
		// Role of subnet within the cluster.
		Role SubnetRole `json:"role" patchStrategy:"merge,retainKeys" patchMergeKey:"role"`
		// The Id of the subnet.
		ID string `json:"id"`
	}
	CommonClusterSpec struct {
		// Kubernetes settings.
		Kubernetes `json:"kubernetes"`
		// Reference for cloud authentication.
		IdentityRef NamespacedRef `json:"identityRef"`
		// Private Registry settings for the workload cluster.
		PrivateRegistry *PrivateRegistry `json:"privateRegistry,omitempty"`
		// HTTP Proxy settings.
		Proxy *Proxy `json:"proxy,omitempty"`
	}
	CommonOCI struct {
		// OCI region where the cluster will be created.
		Region string `json:"region"`
		// OCI Compartment id where the compartment will be created
		Compartment string `json:"compartment"`
		// SSH public key for node ssh.
		SSHPublicKey *string `json:"sshPublicKey,omitempty"`
		// Node image id.
		// The default is the latest OL8 image in the provided compartment.
		ImageId string `json:"imageId"`
		// Cloud-init script to run during node startup.
		CloudInitScript []string `json:"cloudInitScript,omitempty"`
	}
	Kubernetes struct {
		// Kubernetes version.
		Version string `json:"version"`
		// Kubernetes network settings.
		ClusterNetwork ClusterNetwork `json:"clusterNetwork"`
	}
	ClusterNetwork struct {
		// IP range for Kubernetes pods.
		// The default is `10.244.0.0/16`
		PodCIDR string `json:"podCIDR"`
		// IP range for Kubernetes service addresses.
		// The default is `10.96.0.0/16`.
		ServiceCIDR string `json:"serviceCIDR"`
	}
	OCNE struct {
		// OCNE Version.
		Version string `json:"version"`
		// OCNE dependency settings.
		Dependencies OCNEDependencies `json:"dependencies"`
	}
	OCNEDependencies struct {
		// Whether to install OCNE dependencies.
		// The default is `true`.
		Install bool `json:"install"`
	}
	NodeConfig struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name string `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
		// Node pool Shape.
		Shape *string `json:"shape,omitempty"`
		// Number of OCPUs per node, when using flex shapes.
		OCPUs *int `json:"ocpus,omitempty"`
		// Amount of memory per node, in gigabytes, when using flex shapes.
		MemoryGbs *int `json:"memoryGbs,omitempty"`
		// Size of node boot volume, in gigabytes.
		BootVolumeGbs *int `json:"bootVolumeGbs,omitempty"`
		// Number of nodes to create.
		Replicas *int `json:"replicas,omitempty"`
	}
	PrivateRegistry struct {
		// Private registry URL.
		URL string `json:"url"`
		// Reference to private registry credentials secret.
		CredentialsSecret NamespacedRef `json:"credentialSecret"`
	}
	Proxy struct {
		// HTTP Proxy string.
		HTTPProxy string `json:"httpProxy"`
		// HTTPS Proxy string.
		HTTPSProxy string `json:"httpsProxy"`
		// No Proxy string.
		NoProxy string `json:"noProxy"`
	}
	NamespacedRef struct {
		// Name of the ref.
		Name string `json:"name"`
		// Namespace of the ref.
		Namespace string `json:"namespace"`
	}
	QuickCreateStatus struct {
		Phase QuickCreatePhase `json:"phase"`
	}
	QuickCreatePhase string
)

// Subnet Roles
const (
	// SubnetRoleControlPlane is the role of the Control Plane subnet.
	SubnetRoleControlPlane SubnetRole = "control-plane"
	// SubnetRoleControlPlaneEndpoint is the role of the Control Plane endpoint subnet.
	SubnetRoleControlPlaneEndpoint SubnetRole = "control-plane-endpoint"
	// SubnetRoleServiceLB is the role of the load balancer subnet.
	SubnetRoleServiceLB SubnetRole = "service-lb"
	// SubnetRoleWorker is the role of the worker subnet.
	SubnetRoleWorker SubnetRole = "worker"

	// QuickCreatePhaseProvisioning means the Quick Create is in progress.
	QuickCreatePhaseProvisioning QuickCreatePhase = "Provisioning"
	// QuickCreatePhaseComplete means the Quick Create has finished. Quick Create CR cleanup is started once this phase is reached.
	QuickCreatePhaseComplete QuickCreatePhase = "Complete"
)
