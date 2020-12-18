// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package env

import (
	"os"
	"path/filepath"
)

const vzRootDir = "VZ_ROOT_DIR"

// CheckVersionDisabled Disable version checking
const CheckVersionDisabled = "VZ_DISABLE_VERSION_CHECK"

// DisableWebHookValidation Disable webhook validation without removing the webhook itself
const DisableWebHookValidation = "VZ_DISABLE_VALIDATIONS"

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

// IsCheckVersionDisabled If true, perform version checks on upgrade
func IsCheckVersionDisabled() bool {
	return getEnvFunc(CheckVersionDisabled) == "true"
}

// IsValidationDisabled If true, disable the webhook validation logic (webhook will still be active)
func IsValidationDisabled() bool {
	return getEnvFunc(DisableWebHookValidation) == "true"
}
