// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// AuxiliaryImageVolume contains details of an auxiliary image volume
// +k8s:openapi-gen=true
type AuxiliaryImageVolume struct {
	// The emptyDir volume medium. This is an advanced setting that rarely needs to be configured.
	// Defaults to unset, which means the volume's files are stored on the local node's file
	// system for the life of the pod.
	Medium string `json:"medium,omitempty"`

	// The mount path. The files in the path are populated from the same named directory in the
	// images supplied by each container in serverPod.auxiliaryImages. Each volume must be
	// configured with a different mount path. Required.
	MountPath string `json:"mountPath"`

	// The name of the volume. Required.
	Name string `json:"name"`

	// The emptyDir volume size limit. Defaults to unset.
	SizeLimit string `json:"sizeLimit,omitempty"`
}
