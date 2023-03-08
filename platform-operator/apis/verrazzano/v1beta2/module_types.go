// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=verrazzanomodules
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=vzmod;vzmods
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="The current version of the Verrazzano platform."
// +genclient

// VerrazzanoModule specifies a Verrazzano VerrazzanoModule instance
type VerrazzanoModule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleSpec   `json:"spec,omitempty"`
	Status ModuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoModuleList contains a list of Verrazzano Module instance resources.
type VerrazzanoModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoModule `json:"items"`
}

type PlatformRefence struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ModuleSpec defines the attributes for a Verrazzano Module instance
type ModuleSpec struct {
	Enabled         *bool            `json:"enabled,omitempty"`
	Version         *string          `json:"version,omitempty"`
	Chart           *ChartRef        `json:"chart,omitempty"`
	PlatformRef     *PlatformRefence `json:"platformRef,omitempty"`
	TargetNamespace *string          `json:"targetNamespace,omitempty"`
	Reconcile       *bool            `json:"reconcile,omitempty"`
}

type ModuleStateType string

// ModuleStatus defines the observed state of a Verrazzano Module resource.
type ModuleStatus struct {
	// The version of Verrazzano that is installed.
	Version string          `json:"version,omitempty"`
	State   ModuleStateType `json:"state,omitempty"`
	// The latest available observations of an object's current state.
	Conditions []ModuleCondition `json:"conditions,omitempty"`
}

// ModuleConditionType identifies the condition of the Module resource, which can be checked with `kubectl wait`.
type ModuleConditionType string

// ModuleCondition describes the current state of an installation.
type ModuleCondition struct {
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human-readable message indicating details about the last transition.
	Message string `json:"message,omitempty"`
	// Status of the condition: one of `True`, `False`, or `Unknown`.
	Status corev1.ConditionStatus `json:"status"`
	// Type of condition.
	Type ModuleConditionType `json:"type"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoModule{}, &VerrazzanoModuleList{})
}
