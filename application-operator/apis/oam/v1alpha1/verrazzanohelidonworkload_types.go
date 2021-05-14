// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoHelidonWorkloadSpec wraps meta/ObjectMeta & apps/DeploymentSpec.
type VerrazzanoHelidonWorkloadSpec struct {
	// The embedded apps/Deployment
	DeploymentTemplate DeploymentTemplate `json:"deploymentTemplate"`
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
}

// VerrazzanoHelidonWorkloadStatus defines the observed state of VerrazzanoHelidonWorkload
type VerrazzanoHelidonWorkloadStatus struct {
	// The reconcile status of this workload.
	oamrt.ConditionedStatus `json:",inline"`

	// Resources managed by this workload.
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`

	// CurrentUpgradeVersion is the version that was specified when the application was last upgraded with Verrazzano
	CurrentUpgradeVersion string `json:"currentUpgradeVersion,omitempty"`
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
