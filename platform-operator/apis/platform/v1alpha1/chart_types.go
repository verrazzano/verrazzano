// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=opdef;opdefs
// +genclient

// OperatorDefinition specifies a metadata about an operator chart type.
type OperatorDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorDefinitionSpec   `json:"spec,omitempty"`
	Status OperatorDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OperatorDefinitionList contains a list of Platform resources.
type OperatorDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Platform `json:"items"`
}

type ChartRef struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// OperatorDefinitionSpec defines a Verrazzano Platform instance
type OperatorDefinitionSpec struct {
	// Operator lifecycle type, defaults to "helm"
	LifecycleClass  string     `json:"lifecycleClass,omitempty"`
	CRDCharts       []ChartRef `json:"crds,omitempty"`
	CRDDependencies []ChartRef `json:"crdDependencies,omitempty"`
}

type OperatorDefinitionStatus struct {
	// The latest available observations of an object's current state.
	//Conditions []ModuleDefinitionCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=moddef;moddefs
// +genclient

// ModuleDefinition specifies a metadata about a module chart type.
type ModuleDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleDefinitionSpec   `json:"spec,omitempty"`
	Status ModuleDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModuleDefinitionList contains a list of Platform resources.
type ModuleDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Platform `json:"items"`
}

// ModuleDefinitionSpec defines properties of a Module chart type
type ModuleDefinitionSpec struct {
	OperatorDefinition `json:",inline"`
	OperatorCharts     []ChartRef `json:"operators,omitempty"`
	ModuleCharts       []ChartRef `json:"modules,omitempty"`
}

// ModuleDefinitionStatus defines the observed state of a Verrazzano resource.
type ModuleDefinitionStatus struct {
	// The latest available observations of an object's current state.
	//Conditions []ModuleDefinitionCondition `json:"conditions,omitempty"`
	// The version of Verrazzano that is installed.
	//Version string `json:"version,omitempty"`
}

// ModuleDefinitionConditionType identifies the condition of the Platform resource, which can be checked with `kubectl wait`.
type ModuleDefinitionConditionType string

// ModuleDefinitionCondition describes the current state of an installation.
type ModuleDefinitionCondition struct {
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human readable message indicating details about the last transition.
	Message string `json:"message,omitempty"`
	// Status of the condition: one of `True`, `False`, or `Unknown`.
	Status corev1.ConditionStatus `json:"status"`
	// Type of condition.
	Type PlatformConditionType `json:"type"`
}

func init() {
	SchemeBuilder.Register(&OperatorDefinition{}, &OperatorDefinitionList{})
	SchemeBuilder.Register(&ModuleDefinition{}, &ModuleDefinitionList{})
}
