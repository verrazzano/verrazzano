// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StateType identifies the state of an image built request
type StateType string

const (
	// Ready is the state when a ImageBuildRequest resource can perform a build
	Ready StateType = "Ready"

	// Building is the state when an image is being built
	Building StateType = "Building"

	// Published is the state after a successful build of an image
	Published StateType = "Published"

	// Failed is the state when an ImageBuildRequest has failed
	Failed StateType = "Failed"
)

// ImageBuildRequestSpec defines the desired state of ImageBuildRequest
type ImageBuildRequestSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Image to be used as a base image when creating a new image
	BaseImage string `json:"baseImage"`

	// The JDK installer that will be used by the WebLogic Image Tool
	JDKInstaller string `json:"jdkInstaller"`

	// The WebLogic Installer that will be used by the WebLogic Image Tool
	WebLogicInstaller string `json:"webLogicInstaller"`

	// An Image struct that provides more information about the created image
	Image Image `json:"image"`
}

// Image provides more configuration information to the ImageBuildRequestSpec
type Image struct {
	// Name of the image that will be created
	Name string `json:"name"`

	// Tag for the final build image
	Tag string `json:"tag"`

	// Registry to which the image will belong
	Registry string `json:"registry"`

	// Repository to which the image will belong
	Repository string `json:"repository"`
}

// ImageBuildRequestStatus defines the observed state of ImageBuildRequest
type ImageBuildRequestStatus struct {
	// State of the ImageBuildRequest
	State StateType `json:"state,omitempty"`
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
