// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"os"
	"strings"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

var vzMergeStruct vzapi.Verrazzano

// MergeYAMLFiles parses the given slice of filenames containing yaml and
// merges them into a single verrazzano yaml and then returned as a vz resource.
func MergeYAMLFiles(filenames []string) (*vzapi.Verrazzano, error) {
	if len(filenames) == 0 {
		return nil, fmt.Errorf("Failed to merge files - no files specified")
	}
	var vzYaml string
	for _, filename := range filenames {
		readBytes, err := os.ReadFile(strings.TrimSpace(filename))
		if err != nil {
			return nil, err
		}
		vzYaml, err = overlayVerrazzano(vzYaml, string(readBytes))
		if err != nil {
			return nil, err
		}
	}

	vz := &vzapi.Verrazzano{}
	err := yaml.Unmarshal([]byte(vzYaml), &vz)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a verrazzano install resource: %s", err.Error())
	}
	if vz.Namespace == "" {
		vz.Namespace = "default"
	}
	if vz.Name == "" {
		vz.Name = "verrazzano"
	}

	return vz, nil
}

// overlayVerrazzano overlays over base using JSON strategic merge.
func overlayVerrazzano(baseYAML string, overlayYAML string) (string, error) {
	if strings.TrimSpace(baseYAML) == "" {
		return overlayYAML, nil
	}
	if strings.TrimSpace(overlayYAML) == "" {
		return baseYAML, nil
	}
	baseJSON, err := yaml.YAMLToJSON([]byte(baseYAML))
	if err != nil {
		return "", fmt.Errorf("Failed to create a verrazzano install resource: %s\n%s", err.Error(), baseYAML)
	}
	overlayJSON, err := yaml.YAMLToJSON([]byte(overlayYAML))
	if err != nil {
		return "", fmt.Errorf("Failed to create a verrazzano install resource: %s\n%s", err.Error(), overlayYAML)
	}

	// Merge the two json representations
	mergedJSON, err := strategicpatch.StrategicMergePatch(baseJSON, overlayJSON, &vzMergeStruct)
	if err != nil {
		return "", fmt.Errorf("Failed to merge yaml files: %s\n for base object: \n%s\n override object: \n%s", err.Error(), baseJSON, overlayJSON)
	}

	mergedYAML, err := yaml.JSONToYAML(mergedJSON)
	if err != nil {
		return "", fmt.Errorf("Failed to create a verrazzano install resource: %s\n%s", err.Error(), mergedJSON)
	}

	return string(mergedYAML), nil
}
