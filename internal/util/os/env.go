// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import "os"

const vzRootDir = "VZ_ROOT_DIR"

var getEnvFunc func(string) string = os.Getenv

// VzRootDir returns the root director of the verrazzano files on the docker image.
// This can be set by developer to run the operator in development outside of kubernetes
func VzRootDir() string {
	home := getEnvFunc(vzRootDir)
	if len(home) > 0 {
		return home
	}
	return "/verrazzano"
}

