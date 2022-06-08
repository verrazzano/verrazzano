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
// merges them into a single verrazzano yaml which is returned as a string.
func MergeYAMLFiles(filenames []string) (string, error) {
	var vzYaml string
	for _, filename := range filenames {
		readBytes, err := os.ReadFile(strings.TrimSpace(filename))
		if err != nil {
			return "", err
		}
		vzYaml, err = overlayVerrazzano(vzYaml, string(readBytes))
		if err != nil {
			return "", err
		}
	}
	return vzYaml, nil
}

// overlayVerrazzano overlays over base using JSON strategic merge.
func overlayVerrazzano(baseYaml string, overlayYaml string) (string, error) {
	if strings.TrimSpace(baseYaml) == "" {
		return overlayYaml, nil
	}
	if strings.TrimSpace(overlayYaml) == "" {
		return baseYaml, nil
	}
	baseJson, err := yaml.YAMLToJSON([]byte(baseYaml))
	if err != nil {
		return "", fmt.Errorf("YAMLToJSON error in base: %s\n%s", err, baseJson)
	}
	overlayJson, err := yaml.YAMLToJSON([]byte(overlayYaml))
	if err != nil {
		return "", fmt.Errorf("YAMLToJSON error in overlay: %s\n%s", err, overlayJson)
	}

	// Merge the two json representations
	mergedJson, err := strategicpatch.StrategicMergePatch(baseJson, overlayJson, &vzMergeStruct)
	if err != nil {
		return "", fmt.Errorf("json merge error (%v) for base object: \n%s\n override object: \n%s", err, baseJson, overlayJson)
	}

	mergedYaml, err := yaml.JSONToYAML(mergedJson)
	if err != nil {
		return "", fmt.Errorf("JSONToYAML error (%v) for merged object: \n%s", err, mergedJson)
	}

	return string(mergedYaml), nil
}
