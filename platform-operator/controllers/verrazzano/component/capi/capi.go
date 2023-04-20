// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"os"
)

// PreInstall implementation for the CAPI Component
func preInstall(ctx spi.ComponentContext) error {
	// Startup the OCI infrastructure provider without requiring OCI credentials
	os.Setenv("INIT_OCI_CLIENTS_ON_STARTUP", "false")

	return nil
}
