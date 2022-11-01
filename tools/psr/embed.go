// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

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
