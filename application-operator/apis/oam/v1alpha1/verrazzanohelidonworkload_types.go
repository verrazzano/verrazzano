// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoHelidonWorkloadSpec wraps a Helidon application deployment and service.
type VerrazzanoHelidonWorkloadSpec struct {
	// An embedded Helidon application deployment.
	DeploymentTemplate DeploymentTemplate `json:"deploymentTemplate"`
	// An embedded Helidon application service
	ServiceTemplate ServiceTemplate `json:"serviceTemplate,omitempty"`
}

// DeploymentTemplate should have the metadata and spec of the underlying apps/Deployment
type DeploymentTemplate struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata"`
	// The deployment strategy to use to replace existing pods with new ones.
	// +kubebuilder:validation:Optional
	// +patchStrategy=retainKeys
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty" patchStrategy:"retainKeys"  protobuf:"bytes,4,opt,name=strategy"`
	// +kubebuilder:validation:Required
	PodSpec v1.PodSpec `json:"podSpec"`
	// A label selector is a label query over a set of resources
	Selector metav1.LabelSelector `json:"selector,omitempty" patchStrategy:"retainKeys"`
}

// ServiceTemplate specifies the metadata and pod spec of a Helidon workload.
type ServiceTemplate struct {
	// Metadata about a Helidon application.
	// +kubebuilder:validation:Optional
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	// The service spec of a Helidon application.
	// +kubebuilder:validation:Optional
	// +optional
	ServiceSpec v1.ServiceSpec `json:"serviceSpec,omitempty"`
}

// VerrazzanoHelidonWorkloadStatus defines the observed state of Verrazzano Helidon workload.
type VerrazzanoHelidonWorkloadStatus struct {
	// The reconcile status of this workload.
	oamrt.ConditionedStatus `json:",inline"`

	// Resources managed by this workload.
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}

// VerrazzanoHelidonWorkload is the Schema for verrazzanohelidonworkloads API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
type VerrazzanoHelidonWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required
	Spec   VerrazzanoHelidonWorkloadSpec   `json:"spec"`
	Status VerrazzanoHelidonWorkloadStatus `json:"status,omitempty"`
}

// VerrazzanoHelidonWorkloadList contains a list of VerrazzanoHelidonWorkload
// +kubebuilder:object:root=true
type VerrazzanoHelidonWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoHelidonWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoHelidonWorkload{}, &VerrazzanoHelidonWorkloadList{})
}
