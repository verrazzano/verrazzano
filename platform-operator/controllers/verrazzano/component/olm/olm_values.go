// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olm

// olmValues struct representing the Helm chart values for this component
type olmValues struct {
	ImageName    string `json:"imageName,omitempty"`
	ImageVersion string `json:"imageVersion,omitempty"`
}
