// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	vz_os "github.com/verrazzano/verrazzano-platform-operator/internal/util/os"
	"os"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Upgrade will upgrade a Helm release with the specificed charts.
func Upgrade(releaseName string, namespace string, chartDir string) error {
	var log = ctrl.Log.WithName("helm")

//	args := []string{"upgrade", releaseName, chartDir, "--dry-run"}
	args := []string{"upgrade", releaseName, chartDir}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	// todo inject kubeconfig
	configPath := os.Getenv("VERRAZZANO_KUBECONFIG")
	if len(configPath) == 0 {
		configPath = os.Getenv("KUBECONFIG")
	}
	if len(configPath) == 0 {
		configPath = "/home/verrazzano/kubeconfig"
	}

	cmd := exec.Command("helm", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+configPath)
	stdout, stderr, err := vz_os.RunCommand(cmd)
	if err != nil {
		log.Error(err, string(stderr))
		return err
	}

	// Temp log upgrade output
	log.Info(string(stdout))
	return nil
}
