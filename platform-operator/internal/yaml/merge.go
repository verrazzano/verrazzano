// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type PatchStrategy interface{}

// Merge merges the overlay YAML files onto the base YAML file and returns a YAML string.
// The overlay has priority.
func MergeFiles(baseFile string, overlayFile string, strategy PatchStrategy) (string, error) {
	baseYAML, err := ioutil.ReadFile(filepath.Join(baseFile))
	if err != nil {
		return "", err
	}
	overlayYAML, err := ioutil.ReadFile(filepath.Join(overlayFile))
	if err != nil {
		return "", err
	}
	return MergeString(string(baseYAML), string(overlayYAML), strategy)
}

// Merge merges the overlay yaml onto the base yaml. The overlay has priority.
func MergeString(baseYaml string, overlayYaml string, strategy PatchStrategy) (string, error) {
	trimBaseYaml := strings.TrimSpace(baseYaml)
	trimOverlayYaml := strings.TrimSpace(overlayYaml)

	if trimBaseYaml == "" {
		return overlayYaml, nil
	}
	if trimOverlayYaml == "" {
		return baseYaml, nil
	}
	baseJson, err := yaml.YAMLToJSON([]byte(trimBaseYaml))
	if err != nil {
		return "", err
	}
	overlayJson, err := yaml.YAMLToJSON([]byte(trimOverlayYaml))
	if err != nil {
		return "", err
	}
	mergedJson, err := strategicpatch.StrategicMergePatch(baseJson, overlayJson, strategy)
	if err != nil {
		return "", err
	}
	mergedYaml, err := yaml.JSONToYAML([]byte(mergedJson))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(mergedYaml)), nil
}
