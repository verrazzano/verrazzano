// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/types"
)

type (
	SubnetRole string
	Subnet     struct {
		// +patchMergeKey=role
		// +patchStrategy=merge,retainKeys
		// Role of subnet within the cluster.
		Role SubnetRole `json:"role" patchStrategy:"merge,retainKeys" patchMergeKey:"role"`
		// +kubebuilder:validation:Pattern:=`^([0-9a-zA-Z-_]+[.:])([0-9a-zA-Z-_]*[.:]){3,}([0-9a-zA-Z-_]+)$`

		// The ID of the subnet.
		ID string `json:"id"`
	}
	CommonClusterSpec struct {
		// Reference for cloud authentication.
		IdentityRef NamespacedRef `json:"identityRef"`
		// +optional

		// Private Registry settings for the workload cluster.
		PrivateRegistry *PrivateRegistry `json:"privateRegistry,omitempty"`
		// +optional

		// HTTP Proxy settings.
		Proxy *Proxy `json:"proxy,omitempty"`
	}
	CommonOCI struct {
		// OCI region where the cluster will be created.
		Region string `json:"region,omitempty"`
		// +kubebuilder:validation:Pattern:=`^([0-9a-zA-Z-_]+[.:])([0-9a-zA-Z-_]*[.:]){3,}([0-9a-zA-Z-_]+)$`

		// OCI Compartment OCID where the cluster will be created
		Compartment string `json:"compartment,omitempty"`
		// +optional

		// SSH public key for node ssh.
		SSHPublicKey *string `json:"sshPublicKey,omitempty"`
		// +kubebuilder:validation:Pattern:=`^([0-9a-zA-Z-_]+[.:])([0-9a-zA-Z-_]*[.:]){3,}([0-9a-zA-Z-_]+)$`

		// Node image OCID.
		// The default is the latest OL8 image in the provided compartment.
		ImageID string `json:"imageId,omitempty"`
		// +optional

		// Cloud-init script to run during node startup.
		CloudInitScript []string `json:"cloudInitScript,omitempty"`
	}
	Kubernetes struct {
		// +kubebuilder:validation:Pattern:=`^v([0-9]+\.){2}[0-9]+$`

		// Kubernetes version.
		Version        string `json:"version"`
		KubernetesBase `json:",inline"`
	}
	KubernetesBase struct {

		// +kubebuilder:default:={podCIDR: "10.244.0.0/16",serviceCIDR: "10.96.0.0/16"}
		// +optional

		// Kubernetes network settings.
		ClusterNetwork ClusterNetwork `json:"clusterNetwork"`
	}
	ClusterNetwork struct {
		// +kubebuilder:validation:Pattern:=`^([0-9]{1,3}\.){3}[0-9]{1,3}(\/([0-9]|[1-2][0-9]|3[0-2]))$`
		// +optional

		// IP range for Kubernetes pods.
		// The default is `10.244.0.0/16`
		PodCIDR string `json:"podCIDR"`
		// +kubebuilder:validation:Pattern:=`^([0-9]{1,3}\.){3}[0-9]{1,3}(\/([0-9]|[1-2][0-9]|3[0-2]))$`
		// +optional

		// IP range for Kubernetes service addresses.
		// The default is `10.96.0.0/16`.
		ServiceCIDR string `json:"serviceCIDR"`
	}
	OCNE struct {
		// OCNE Version.
		Version string `json:"version"`
		// +kubebuilder:default:={skipInstall: false}
		// +optional

		// OCNE dependency settings.
		Dependencies OCNEDependencies `json:"dependencies"`
	}
	OCNEDependencies struct {
		// +optional
		// +kubebuilder:default:=false

		// Whether to skip OCNE dependency installation.
		// The default is `false`.
		SkipInstall bool `json:"skipInstall"`
	}
	NamedOCINode struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name    string `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
		OCINode `json:",inline"`
	}
	OCINode struct {
		// Node pool Shape.
		Shape *string `json:"shape"`
		// +optional
		// +kubebuilder:validation:Minimum:=1
		// +kubebuilder:validation:Maximum:=999

		// Number of OCPUs per node, when using flex shapes.
		OCPUs *int `json:"ocpus,omitempty"`
		// +kubebuilder:validation:Minimum:=1
		// +kubebuilder:validation:Maximum:=999

		// Amount of memory per node, in gigabytes, when using flex shapes.
		MemoryGbs *int `json:"memoryGbs,omitempty"`
		// +optional
		// +kubebuilder:validation:Minimum:=50
		// +kubebuilder:validation:Maximum:=32000

		// Size of node boot volume, in gigabytes.
		BootVolumeGbs *int `json:"bootVolumeGbs,omitempty"`
		// +kubebuilder:validation:Minimum:=1
		// +kubebuilder:validation:Maximum:=999

		// Number of nodes to create.
		Replicas *int `json:"replicas"`
	}
	PrivateRegistry struct {
		// Private registry URL.
		URL string `json:"url"`
		// +optional

		// Reference to private registry credentials secret.
		CredentialsSecret NamespacedRef `json:"credentialSecret"`
	}
	Proxy struct {
		// HTTP Proxy string.
		HTTPProxy string `json:"httpProxy"`
		// HTTPS Proxy string.
		HTTPSProxy string `json:"httpsProxy"`
		// +optional

		// No Proxy string.
		NoProxy string `json:"noProxy"`
	}
	NamespacedRef struct {
		// +kubebuilder:validation:MaxLength:=63
		// +kubebuilder:validation:MinLength:=1

		// Name of the ref.
		Name string `json:"name"`
		// +kubebuilder:validation:MaxLength:=63
		// +kubebuilder:validation:MinLength:=1

		// Namespace of the ref.
		Namespace string `json:"namespace"`
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

func (n NamespacedRef) AsNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: n.Namespace,
		Name:      n.Name,
	}
}
