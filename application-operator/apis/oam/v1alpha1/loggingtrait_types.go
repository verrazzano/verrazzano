// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LoggingTraitKind is the Kind of the LoggingTrait.
const LoggingTraitKind string = "LoggingTrait"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LoggingTraitSpec defines the desired state of LoggingTrait
type LoggingTraitSpec struct {
	// LoggingConfig represents the configuration provided by the user for the FluentD configuration that consists of fluentd.conf: "<source>\n ... and so on ...\n"
	LoggingConfig string `json:"loggingConfig,omitempty"`

	// LoggingImage represents the optional FluentD image provided by the user
	// +optional
	LoggingImage string `json:"loggingImage,omitempty"`

	// ImagePullPolicy represents the optional image pull policy for the FluentD image provided by the user
	// +optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// The WorkloadReference to the workload to which this trait applies.
	// This value is populated by the OAM runtime when a ApplicationConfiguration
	// resource is processed.  When the ApplicationConfiguration is processed a trait and
	// a workload resource are created from the content of the ApplicationConfiguration.
	// The WorkloadReference is provided in the trait by OAM to ensure the trait controller
	// can find the workload associated with the component containing the trait within the
	// original ApplicationConfiguration.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// LoggingTraitStatus defines the observed state of LoggingTrait
type LoggingTraitStatus struct {
	// The reconcile status of this logging trait
	oamrt.ConditionedStatus `json:",inline"`
	// The resources managed by this logging trait
	Resources []oamrt.TypedReference `json:"resources,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:object:generate=true

// LoggingTrait is the Schema for the loggingtraits API
type LoggingTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoggingTraitSpec   `json:"spec,omitempty"`
	Status LoggingTraitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true
// LoggingTraitList contains a list of LoggingTrait
type LoggingTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoggingTrait `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoggingTrait{}, &LoggingTraitList{})
}
