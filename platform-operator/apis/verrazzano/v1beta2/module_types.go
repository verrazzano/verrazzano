// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=modules,shortName=vzmod;vzmods
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="The current version of the Verrazzano platform."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state",description="State of Module reconciliation"
// +genclient

// Module specifies a Verrazzano Module instance
type Module struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModuleSpec   `json:"spec,omitempty"`
	Status ModuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModuleList contains a list of Verrazzano Module instance resources.
type ModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Module `json:"items"`
}

// ModuleSpec defines the attributes for a Verrazzano Module instance
type ModuleSpec struct {
	Name            string        `json:"name,omitempty"`
	Enabled         bool          `json:"enabled,omitempty"`
	Version         string        `json:"version,omitempty"`
	TargetNamespace string        `json:"targetNamespace,omitempty"`
	Reconcile       bool          `json:"reconcile,omitempty"`
	Source          *ModuleSource `json:"source,omitempty"`
	Overrides       []Overrides   `json:"overrides,omitempty"`
}

// Overrides identifies overrides for a component.
type Overrides struct {
	// Selector for ConfigMap containing override data.
	// +optional
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
	// Selector for Secret containing override data.
	// +optional
	SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`
	// Configure overrides using inline YAML.
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}

type ModuleStateType string

const (
	ModuleStateUnknown     = "Unknown"
	ModuleStateReconciling = "Reconciling"
	ModuleStateReady       = "Ready"
)

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

type ChartVersion struct {
	Name              string `json:"name"`
	DefaultVersion    string `json:"defaultVersion,omitempty"`
	SupportedVersions string `json:"supportedVersions,omitempty"`
}

type HelmChartRepository struct {
	Name              string `json:"name"`
	URI               string `json:"uri"`
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

type ModuleSourceRef struct {
}

type ModuleSource struct {
	ChartRepo *HelmChartRepository `json:"repo,omitempty"`
	SourceRef *ModuleSourceRef     `json:"sourceRef,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Module{}, &ModuleList{})
}
