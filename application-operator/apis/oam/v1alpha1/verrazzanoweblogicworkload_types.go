// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VerrazzanoWebLogicWorkloadTemplate struct {
	APIVersion string               `json:"apiVersion,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata   runtime.RawExtension `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Spec runtime.RawExtension `json:"spec"`
}

// VerrazzanoWebLogicWorkloadSpec wraps a WebLogic resource. The WebLogic domain specified
// in the template must contain a spec field and it may include a metadata field.
type VerrazzanoWebLogicWorkloadSpec struct {
	// The metadata and spec for the underlying
	// <a href="https://github.com/oracle/weblogic-kubernetes-operator/blob/main/documentation/domains/Domain.md">Domain</a> resource.
	// +kubebuilder:pruning:PreserveUnknownFields
	Template VerrazzanoWebLogicWorkloadTemplate   `json:"template"`
	Clusters []VerrazzanoWebLogicWorkloadTemplate `json:"clusters,omitempty"`
}

// VerrazzanoWebLogicWorkloadStatus defines the observed state of a Verrazzano WebLogic workload.
type VerrazzanoWebLogicWorkloadStatus struct {
	// The last generation of the Verrazzano WebLogic workload that was reconciled.
	LastGeneration string `json:"lastGeneration,omitempty"`
	// The last value of the `verrazzano.io/restart-version` annotation.
	LastRestartVersion string `json:"lastRestartVersion,omitempty"`
	// The last value of the `verrazzano.io/lifecycle-action`.
	LastLifecycleAction string `json:"lastLifecycleAction,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VerrazzanoWebLogicWorkload specifies the Verrazzano WebLogic workload API.
type VerrazzanoWebLogicWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Verrazzano WebLogic workload.
	Spec VerrazzanoWebLogicWorkloadSpec `json:"spec,omitempty"`
	// The observed state of a Verrazzano WebLogic workload.
	Status VerrazzanoWebLogicWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoWebLogicWorkloadList contains a list of Verrazzano WebLogic workload resources.
type VerrazzanoWebLogicWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoWebLogicWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoWebLogicWorkload{}, &VerrazzanoWebLogicWorkloadList{})
}
