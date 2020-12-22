// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	vz_os "github.com/verrazzano/verrazzano/operator/internal/util/os"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
)

// cmdRunner needed for unit tests
var runner vz_os.CmdRunner = vz_os.DefaultRunner{}

// Upgrade will upgrade a Helm release with the specificed charts.
func Upgrade(releaseName string, namespace string, chartDir string) (stdout []byte, stderr []byte, err error) {
	var log = ctrl.Log.WithName("helm")

	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"upgrade", releaseName, chartDir}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	cmd := exec.Command("helm", args...)
	stdout, stderr, err = runner.Run(cmd)
	if err != nil {
		log.Error(err, fmt.Sprintf("Verrazzano helm upgrade failed with stderr: %s\n", string(stderr)))
		return stdout, stderr, err
	}

	//  Log upgrade output
	log.Info(fmt.Sprintf("Verrazzano helm upgrade succeeded with stdout: %s\n", string(stdout)))
	return stdout, stderr, nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vz_os.CmdRunner) {
	runner = r
}
