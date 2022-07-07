// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	osexec "os/exec"

	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext) error {
	ctx.Log().Oncef("Running the Rancher uninstall system tool")

	cmd := osexec.Command("Rancher uninstall binary", "/usr/local/bin/system-tools", "remove", "-c", "/home/verrazzano/kubeconfig", "--force")
	_, stdErr, err := os.DefaultRunner{}.Run(cmd)
	if err != nil {
		return ctx.Log().ErrorNewErr("Failed to run system tools  for Rancher deletion: %s: %v", stdErr, err)
	}

	return nil
}
