// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const VerrazzanoProjectKind = "VerrazzanoProject"
const VerrazzanoProjectResource = "verrazzanoprojects"

// NamespaceTemplate contains the metadata and specification of a Kubernetes namespace.
type NamespaceTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata"`
	// The specification of a namespace.
	Spec corev1.NamespaceSpec `json:"spec,omitempty"`
}

// NetworkPolicyTemplate contains the metadata and specification of a Kubernetes NetworkPolicy.
type NetworkPolicyTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata"`
	// The specification of a network policy.
	Spec netv1.NetworkPolicySpec `json:"spec,omitempty"`
}

// SecuritySpec defines the security configuration for a Verrazzano Project.
type SecuritySpec struct {
	// The subjects to bind to the `verrazzano-project-admin` role.
	// +optional
	ProjectAdminSubjects []rbacv1.Subject `json:"projectAdminSubjects,omitempty"`
	// The subjects to bind to the `verrazzano-project-monitoring` role.
	// +optional
	ProjectMonitorSubjects []rbacv1.Subject `json:"projectMonitorSubjects,omitempty"`
}

// ProjectTemplate contains the list of namespaces to create and the optional security configuration for each namespace.
type ProjectTemplate struct {
	// The list of application namespaces to create for this project.
	Namespaces []NamespaceTemplate `json:"namespaces"`

	// Network policies applied to namespaces in the project.
	// +optional
	NetworkPolicies []NetworkPolicyTemplate `json:"networkPolicies,omitempty"`

	// The project security configuration.
	// +optional
	Security SecuritySpec `json:"security,omitempty"`
}

// VerrazzanoProjectSpec defines the desired state of a Verrazzano Project.
type VerrazzanoProjectSpec struct {
	// Clusters on which the namespaces are to be created.
	Placement Placement `json:"placement"`

	// The project template.
	Template ProjectTemplate `json:"template"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=vp;vps
// +kubebuilder:subresource:status

// VerrazzanoProject specifies the Verrazzano Projects API.
type VerrazzanoProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Verrazzano Project resource.
	Spec VerrazzanoProjectSpec `json:"spec"`
	// The observed state of a Verrazzano Project resource.
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoProjectList contains a list of Verrazzano Project resources.
type VerrazzanoProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoProject{}, &VerrazzanoProjectList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource.
func (in *VerrazzanoProject) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetPlacement returns the Placement of this resource.
func (in *VerrazzanoProject) GetPlacement() Placement {
	return in.Spec.Placement
}
