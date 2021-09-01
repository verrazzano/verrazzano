// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// AuxiliaryImage contains details of an auxiliary image
// +k8s:openapi-gen=true
type AuxiliaryImage struct {
	// The command for this init container. Defaults to cp -R $AUXILIARY_IMAGE_PATH/* $TARGET_MOUNT_PATH. This is
	// an advanced setting for customizing the container command for copying files from the container image to the
	// auxiliary image emptyDir volume. Use the $AUXILIARY_IMAGE_PATH environment variable to reference the value
	// configured in spec.auxiliaryImageVolumes.mountPath, which defaults to "/auxiliary". Use '$TARGET_MOUNT_PATH'
	// to refer to the temporary directory created by the operator that resolves to the auxiliary image's internal
	// emptyDir volume.
	Command string `json:"command,omitempty"`

	// The name of an image with files located in the directory specified by spec.auxiliaryImageVolumes.mountPath
	// of the auxiliary image volume referenced by serverPod.auxiliaryImages.volume, which defaults to "/auxiliary".
	Image string `json:"image,omitempty"`

	// The image pull policy for the container image. Legal values are Always, Never, and IfNotPresent. Defaults to
	// Always if image ends in :latest; IfNotPresent, otherwise.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// The name of an auxiliary image volume defined in spec.auxiliaryImageVolumes. Required.
	Volume string `json:"volume"`
}
