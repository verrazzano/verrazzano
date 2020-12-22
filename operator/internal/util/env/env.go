// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package env

import (
	"os"
	"path/filepath"
)

const vzRootDir = "VZ_ROOT_DIR"

// CheckVersionEnabled Disable version checking
const CheckVersionEnabled = "VZ_CHECK_VERSION"

// WebHookValidationEnabled Disable webhook validation without removing the webhook itself
const WebHookValidationEnabled = "VZ_VALIDATION_ENABLED"

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

// IsVersionCheckEnabled If true, perform version checks on upgrade
func IsVersionCheckEnabled() bool {
	return getEnvFunc(CheckVersionEnabled) != "false"
}

// IsValidationEnabled If true, enable the webhook validation logic
func IsValidationEnabled() bool {
	return getEnvFunc(WebHookValidationEnabled) != "false"
}
