// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package k8sutil

import (
	"fmt"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"os/exec"
	"strings"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// GetInstalledBOMData Exec's into the Platform Operator pod and returns the installed BOM file data as JSON
func GetInstalledBOMData(kubeconfigPath string) ([]byte, error) {
	const platformOperatorPodNameSearchString = "verrazzano-platform-wls" // Pod Substring for finding the platform wls pod

	kubeconfigArgs := []string{}
	if len(kubeconfigPath) > 0 {
		kubeconfigArgs = append(kubeconfigArgs, "--kubeconfig", kubeconfigPath)
	}

	listPodsArgs := []string{"get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install"}
	if len(kubeconfigArgs) > 0 {
		listPodsArgs = append(listPodsArgs, kubeconfigArgs...)
	}
	cmd := exec.Command("kubectl", listPodsArgs...)
	podListOutput, _, err := runner.Run(cmd)
	if err != nil {
		return []byte{}, err
	}
	var platformOperatorPodName = ""
	vzInstallPods := string(podListOutput)
	vzInstallPodArray := strings.Split(vzInstallPods, "\n")
	for _, podName := range vzInstallPodArray {
		if strings.Contains(podName, platformOperatorPodNameSearchString) {
			platformOperatorPodName = podName
			break
		}
	}
	if platformOperatorPodName == "" {
		return []byte{}, fmt.Errorf("pod not found in verrazzano-install namespace")
	}

	platformOperatorPodName = strings.TrimSuffix(platformOperatorPodName, "\n")

	//  Get the BOM from platform-wls
	getBOMArgs := []string{"exec", "-it", platformOperatorPodName, "-n", "verrazzano-install", "--", "cat", "/verrazzano/platform-wls/verrazzano-bom.json"}
	if len(kubeconfigPath) > 0 {
		getBOMArgs = append(getBOMArgs, "--kubeconfig", kubeconfigPath)
	}
	cmd = exec.Command("kubectl", getBOMArgs...)
	bomBytes, _, err := runner.Run(cmd)
	if err != nil {
		return []byte{}, err
	}
	if len(bomBytes) == 0 {
		return bomBytes, fmt.Errorf("Error retrieving BOM from platform wls, no data found")
	}
	return bomBytes, nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
