// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=platforms
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=pf;pfs
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="The current version of the Verrazzano platform."
// +genclient

// Platform specifies a Verrazzano Platform instance.
type Platform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformSpec   `json:"spec,omitempty"`
	Status PlatformStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlatformList contains a list of Platform resources.
type PlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Platform `json:"items"`
}

type ChartVersion struct {
	Name         string `json:"name"`
	VersionRange string `json:"versions"`
}

// PlatformSpec defines valid versions for a Verrazzano Platform instance
type PlatformSpec struct {
	Version          string         `json:"version"`
	CRDVersions      []ChartVersion `json:"crds,omitempty"`
	OperatorVersions []ChartVersion `json:"operators,omitempty"`
	ModuleVersions   []ChartVersion `json:"modules,omitempty"`
}

// PlatformStatus defines the observed state of a Verrazzano resource.
type PlatformStatus struct {
	// The latest available observations of an object's current state.
	//Conditions []PlatformCondition `json:"conditions,omitempty"`
	// The version of Verrazzano that is installed.
	Version string `json:"version,omitempty"`
}

// PlatformConditionType identifies the condition of the Platform resource, which can be checked with `kubectl wait`.
type PlatformConditionType string

// PlatformCondition describes the current state of an installation.
type PlatformCondition struct {
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human-readable message indicating details about the last transition.
	Message string `json:"message,omitempty"`
	// Status of the condition: one of `True`, `False`, or `Unknown`.
	Status corev1.ConditionStatus `json:"status"`
	// Type of condition.
	Type PlatformConditionType `json:"type"`
}

func init() {
	SchemeBuilder.Register(&Platform{}, &PlatformList{})
}
