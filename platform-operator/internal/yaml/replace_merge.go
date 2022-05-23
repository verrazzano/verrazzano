// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"sigs.k8s.io/yaml"
	"strings"
)

// ReplacementMerge merges the YAML files returns a YAML string.
// The first YAML is overlayed by each subsequent YAML, lists are replaced
func ReplacementMerge(yamls ...string) (string, error) {
	if len(yamls) == 0 {
		return "", nil
	}
	if len(yamls) == 1 {
		return yamls[0], nil
	}
	var mBase = yamlMap{}
	for _, yam := range yamls {
		if len(mBase.yMap) == 0 {
			if err := yaml.Unmarshal([]byte(yam), &mBase.yMap); err != nil {
				return "", err
			}
			continue
		}
		var mOverlay = yamlMap{}
		if err := yaml.Unmarshal([]byte(yam), &mOverlay.yMap); err != nil {
			return "", err
		}
		if err := MergeMaps(mBase.yMap, mOverlay.yMap); err != nil {
			return "", err
		}
	}
	b, err := yaml.Marshal(&mBase.yMap)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// mergeMaps 2 maps where mOverlay overrides mBase
func MergeMaps(mBase map[string]interface{}, mOverlay map[string]interface{}) error {
	for k, vOverlay := range mOverlay {
		vBase, ok := mBase[k]
		recursed := false
		if ok {
			// Both mBase and mOverlay have this key. If these are nested maps merge them
			switch tBase := vBase.(type) {
			case map[string]interface{}:
				switch tOverlay := vOverlay.(type) {
				case map[string]interface{}:
					MergeMaps(tBase, tOverlay)
					recursed = true
				}
			}
		}
		// Both values were not maps, put overlay entry into the base map
		// This might be a new base entry or replaced by the mOverlay value
		if !recursed {
			mBase[k] = vOverlay
		}
	}
	return nil
}
