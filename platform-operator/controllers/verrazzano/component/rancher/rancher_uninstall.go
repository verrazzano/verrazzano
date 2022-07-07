// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext) error {
	_, stdErr, err := os.RunBash("/usr/local/bin/system-tools", "remove", "-c", "/home/verrazzano/kubeconfig", "--force")
	if err != nil {
		return ctx.Log().ErrorNewErr("Failed to run system tools  for Rancher deletion: %s: %v", stdErr, err)
	}
	return nil
}
