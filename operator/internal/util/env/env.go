// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package env

import (
	"os"
	"path/filepath"
)

const vzRootDir = "VZ_ROOT_DIR"

const checkVersion = "VZ_CHECK_VERSION"

var getEnvFunc func(string) string = os.Getenv

// VzRootDir returns the root directory of the verrazzano files on the docker image.
// This can be set by developer to run the operator in development outside of kubernetes
func VzRootDir() string {
	home := getEnvFunc(vzRootDir)
	if len(home) > 0 {
		return home
	}
	return "/verrazzano"
}

// VzChartDir returns the chart directory of the verrazzano helm chart on the docker image.
// This can be set by developer to run the operator in development outside of kubernetes
func VzChartDir() string {
	home := getEnvFunc(vzRootDir)
	if len(home) > 0 {
		return filepath.Join(home + "/operator/scripts/install/chart")
	}
	return "/verrazzano/install/chart"
}

// IsCheckVersionRequired If true, perform version checks on upgrade
func IsCheckVersionRequired() bool {
	return getEnvFunc(checkVersion) != "false"
}
