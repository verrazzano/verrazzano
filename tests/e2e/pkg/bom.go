// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
)

const (
	// Pod Substring for finding the platform operator pod
	platformOperatorPodNameSearchString = "verrazzano-platform-operator"
)

// Get the BOM from the platform operator in the cluster and build the BOM structure from it
func GetBOMDoc() (*bom.BomDoc, error) {
	var platformOperatorPodName = ""

	out, err := exec.Command("kubectl", "get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install").Output()
	if err != nil {
		return nil, fmt.Errorf("error in gettting %s pod name: %v", platformOperatorPodNameSearchString, err)
	}
	vzInstallPods := string(out)
	vzInstallPodArray := strings.Split(vzInstallPods, "\n")
	for _, podName := range vzInstallPodArray {
		if strings.Contains(podName, platformOperatorPodNameSearchString) {
			platformOperatorPodName = podName
			break
		}
	}
	if platformOperatorPodName == "" {
		return nil, fmt.Errorf("platform operator pod name not found in verrazzano-install namespace")
	}

	platformOperatorPodName = strings.TrimSuffix(platformOperatorPodName, "\n")
	fmt.Printf("Getting the registry details in BOM from the platform operator %s\n", platformOperatorPodName)

	// Get the BOM from platform-operator
	out, err = exec.Command("kubectl", "exec", "-it", platformOperatorPodName, "-n", "verrazzano-install", "--",
		"cat", "/verrazzano/platform-operator/verrazzano-bom.json").Output()
	if err != nil {
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, fmt.Errorf("error retrieving BOM from platform operator, zero length")
	}
	var bomDoc bom.BomDoc
	err = json.Unmarshal(out, &bomDoc)
	return &bomDoc, err
}
