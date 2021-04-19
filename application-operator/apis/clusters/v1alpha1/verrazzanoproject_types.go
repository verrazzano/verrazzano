// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceTemplate has the metadata and spec of the underlying namespace
type NamespaceTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta    `json:"metadata"`
	Spec     corev1.NamespaceSpec `json:"spec,omitempty"`
}

// NetworkPolicyTemplate has the metadata and spec of the underlying NetworkPolicy
type NetworkPolicyTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata"`

	// NetworkPolicySpec specifies the NetworkPolicy for a specific namespace / pod combination.
	Spec netv1.NetworkPolicySpec `json:"spec,omitempty"`
}

// SecuritySpec defines the security configuration for a project
type SecuritySpec struct {
	// ProjectAdminSubjects specifies the list of subjects that should be bound to the verrazzano-project-admin role
	// +optional
	ProjectAdminSubjects []rbacv1.Subject `json:"projectAdminSubjects,omitempty"`
	// ProjectMonitorBinding specifies the subject that should be bound to the verrazzano-project-monitor role
	// +optional
	ProjectMonitorSubjects []rbacv1.Subject `json:"projectMonitorSubjects,omitempty"`
}

// ProjectTemplate contains the resources for a project
type ProjectTemplate struct {
	Namespaces []NamespaceTemplate `json:"namespaces"`

	// Security specifies the project security configuration
	// +optional
	Security SecuritySpec `json:"security,omitempty"`

	// Network policies applied to namespaces in the project
	// +optional
	NetworkPolicies []NetworkPolicyTemplate `json:"networkPolicies,omitempty"`
}

// VerrazzanoProjectSpec defines the desired state of VerrazzanoProject
type VerrazzanoProjectSpec struct {
	Template ProjectTemplate `json:"template"`

	// Clusters in which the namespaces in this project are to be placed
	Placement Placement `json:"placement"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=vp;vps
// +kubebuilder:subresource:status

// VerrazzanoProject is the Schema for the verrazzanoprojects API
type VerrazzanoProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoProjectSpec      `json:"spec"`
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoProjectList contains a list of VerrazzanoProject
type VerrazzanoProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoProject{}, &VerrazzanoProjectList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource
func (in *VerrazzanoProject) GetStatus() MultiClusterResourceStatus {
	return in.Status
}
