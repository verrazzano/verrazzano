// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type PatchStrategy interface{}

type yamlMap struct {
	yMap map[string]interface{}
}

// StrategicMergeFiles merges the YAML files returns a YAML string.
// The first file is overlayed by each subsequent file
// The strategy is a JSON annotated structure that represents the YAML structure
func StrategicMergeFiles(strategy PatchStrategy, yamlFiles ...string) (string, error) {
	var yamls []string
	for _, f := range yamlFiles {
		yam, err := os.ReadFile(filepath.Join(f))
		if err != nil {
			return "", err
		}
		yamls = append(yamls, string(yam))
	}
	return StrategicMerge(strategy, yamls...)
}

// StrategicMerge merges the YAML files returns a YAML string.
// The first YAML is overlayed by each subsequent YAML
// The strategy is a JSON annotated structure that represents the YAML structure
func StrategicMerge(strategy PatchStrategy, yamls ...string) (string, error) {
	if len(yamls) == 0 {
		return "", errors.New("At least 1 YAML file is required")
	}
	if len(yamls) == 1 {
		return yamls[0], nil
	}

	var mergedJSON []byte
	for _, yam := range yamls {
		// First time through create the base JSON
		overlayYAML := strings.TrimSpace(yam)
		overlayJSON, err := yaml.YAMLToJSON([]byte(overlayYAML))
		if err != nil {
			return "", err
		}
		if len(mergedJSON) == 0 {
			mergedJSON = overlayJSON
			continue
		}
		mergedJSON, err = strategicpatch.StrategicMergePatch(mergedJSON, overlayJSON, strategy)
		if err != nil {
			return "", err
		}
	}
	mergedYaml, err := yaml.JSONToYAML(mergedJSON)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(mergedYaml)), nil
}
