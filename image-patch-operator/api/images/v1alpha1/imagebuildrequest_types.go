// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageBuildRequestSpec defines the desired state of ImageBuildRequest
type ImageBuildRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	BaseImage         string `json:"baseImage,omitempty"`
	JDKInstaller      string `json:"jdkInstaller"`
	WebLogicInstaller string `json:"webLogicInstaller"`

	Image Image `json:"image"`
}

type Image struct {
	Name       string `json:"name,omitempty"`
	Tag        string `json:"tag"`
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
}

// ImageBuildRequestStatus defines the observed state of ImageBuildRequest
type ImageBuildRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
// +kubebuilder:resource:shortName=ibr;ibrs
//+kubebuilder:subresource:status
// +genclient

// ImageBuildRequest is the Schema for the imagebuildrequests API
type ImageBuildRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageBuildRequestSpec   `json:"spec,omitempty"`
	Status ImageBuildRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageBuildRequestList contains a list of ImageBuildRequest
type ImageBuildRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageBuildRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageBuildRequest{}, &ImageBuildRequestList{})
}
