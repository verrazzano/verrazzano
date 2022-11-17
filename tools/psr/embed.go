// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package psr is ued to reference the embedded manifest files in the binary.
// The embed.go file needs to be in an ancestor directory of the psr/manifests directory or the code
// will not be able to access the manifests.
package psr

import (
	"embed"
)

//go:embed manifests
var manifests embed.FS

// GetEmbeddedManifests returns the embedded manifests
func GetEmbeddedManifests() embed.FS {
	return manifests
}
