// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	osexec "os/exec"
	"regexp"
)

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext) error {
	ctx.Log().Oncef("Running the Rancher uninstall system tool")

	// List all the namespaces that need to be cleaned from Rancher components
	nsList := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &nsList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the Rancher namespaces: %v", err)
	}

	// For Rancher namespaces, run the system tools uninstaller
	for _, ns := range nsList.Items {
		matches, err := regexp.MatchString("^cattle-|^local|^p-|^user-|^fleet|^rancher", ns.Name)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to verify that namespace %s is a Rancher namespace: %v", ns.Name, err)
		}
		if matches {
			args := []string{"/usr/local/bin/system-tools", "remove", "-c", "/home/verrazzano/kubeconfig", "--namespace", ns.Name, "--force"}
			cmd := osexec.Command("Rancher uninstall binary", args...) //nolint:gosec //#nosec G204
			_, stdErr, err := os.DefaultRunner{}.Run(cmd)
			if err != nil {
				return ctx.Log().ErrorNewErr("Failed to run system tools for Rancher deletion: %s: %v", stdErr, err)
			}
		}
	}
	return nil
}
