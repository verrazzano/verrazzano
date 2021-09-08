// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type PatchStrategy interface{}

// MergeFiles merges the YAML files returns a YAML string.
// The first file is overlayed by each subsequent file
// The strategy is a JSON annotated structure that represents the YAML structure
func MergeFiles(strategy PatchStrategy, yamlFiles ...string) (string, error) {
	var yamls []string
	for _, f := range yamlFiles {
		yam, err := ioutil.ReadFile(filepath.Join(f))
		if err != nil {
			return "", err
		}
		yamls = append(yamls, string(yam))
	}
	return MergeString(strategy, yamls...)
}

// MergeString merges the YAML files returns a YAML string.
// The first YAML is overlayed by each subsequent YAML
// The strategy is a JSON annotated structure that represents the YAML structure
func MergeString(strategy PatchStrategy, yamls ...string) (string, error) {
	if len(yamls) == 0 {
		return "", errors.New("At least 1 YAML file is required")
	}
	if len(yamls) == 1 {
		return yamls[0], nil
	}

	var mergedJson []byte
	for _, yam := range yamls {
		// First time through create the base JSON
		overlayYaml := strings.TrimSpace(yam)
		overlayJson, err := yaml.YAMLToJSON([]byte(overlayYaml))
		if err != nil {
			return "", err
		}
		if len(mergedJson) == 0 {
			mergedJson = overlayJson
			continue
		}
		mergedJson, err = strategicpatch.StrategicMergePatch(mergedJson, overlayJson, strategy)
		if err != nil {
			return "", err
		}
	}
	mergedYaml, err := yaml.JSONToYAML([]byte(mergedJson))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(mergedYaml)), nil
}
