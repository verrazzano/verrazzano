// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoHelidonWorkloadSpec wraps a Helidon application deployment.
type VerrazzanoHelidonWorkloadSpec struct {
	// An embedded Helidon application deployment.
	DeploymentTemplate DeploymentTemplate `json:"deploymentTemplate"`
}

// DeploymentTemplate specifies the metadata and pod spec of a Helidon workload.
type DeploymentTemplate struct {
	// Metadata about a Helidon application.
	// +kubebuilder:validation:Required
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata"`
	// The pod spec of a Helidon application.
	// +kubebuilder:validation:Required
	PodSpec v1.PodSpec `json:"podSpec"`
	// Label selector of a Helidon application.
	// +optional
	Selector metav1.LabelSelector `json:"selector,omitempty" patchStrategy:"retainKeys"`
	// The replacement strategy of a Helidon application.
	// +kubebuilder:validation:Optional
	// +patchStrategy=retainKeys
	// +optional
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty" patchStrategy:"retainKeys"  protobuf:"bytes,4,opt,name=strategy"`
}

// VerrazzanoHelidonWorkloadStatus defines the observed state of Verrazzano Helidon workload.
type VerrazzanoHelidonWorkloadStatus struct {
	// Reconcile status of this Verrazzano Helidon workload.
	oamrt.ConditionedStatus `json:",inline"`

	// The resources managed by this Verrazzano Helidon workload.
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VerrazzanoHelidonWorkload specifies the Verrazzano Helidon workload API.
type VerrazzanoHelidonWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// The desired state of a Verrazzano Helidon workload.
	// +kubebuilder:validation:Required
	Spec VerrazzanoHelidonWorkloadSpec `json:"spec"`
	// The observed state of a Verrazzano Helidon workload.
	Status VerrazzanoHelidonWorkloadStatus `json:"status,omitempty"`
}

// VerrazzanoHelidonWorkloadList contains a list of Verrazzano Helidon workload resources.
// +kubebuilder:object:root=true
type VerrazzanoHelidonWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoHelidonWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoHelidonWorkload{}, &VerrazzanoHelidonWorkloadList{})
}
