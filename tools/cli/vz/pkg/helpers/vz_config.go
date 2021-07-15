// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"errors"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

// Helper function to obtain the default kubeConfig location
func GetVZConfigLocation() (string, error) {

	var vzConfig string
	vzConfigEnvVar := os.Getenv("VZCONFIG")

	if len(vzConfigEnvVar) > 0 {
		// Find using environment variables
		vzConfig = vzConfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		vzConfig = filepath.Join(home, ".verrazzano", "config")
	} else {
		// give up
		return "", errors.New("Could not find vz config")
	}
	return vzConfig, nil
}