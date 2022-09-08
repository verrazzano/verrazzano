// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

// MergeYAMLFiles parses the given slice of filenames containing yaml and
// merges them into a single verrazzano yaml and then returned as a vz resource.
func MergeYAMLFiles(filenames []string, stdinReader io.Reader) (*unstructured.Unstructured, error) {
	var vzYAML string
	var stdin bool
	var gv = &schema.GroupVersion{}
	for _, filename := range filenames {
		var readBytes []byte
		var err error
		// filename of "-" is for reading from stdin
		if filename == "-" {
			if stdin {
				continue
			}
			stdin = true
			readBytes, err = io.ReadAll(stdinReader)
		} else {
			readBytes, err = os.ReadFile(strings.TrimSpace(filename))
		}
		if err != nil {
			return nil, err
		}
		if err := checkGroupVersion(readBytes, gv); err != nil {
			return nil, err
		}
		vzYAML, err = overlayVerrazzano(*gv, vzYAML, string(readBytes))
		if err != nil {
			return nil, err
		}
	}

	vz := &unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(vzYAML), &vz)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a verrazzano install resource: %s", err.Error())
	}
	if vz.GetNamespace() == "" {
		vz.SetNamespace("default")
	}
	if vz.GetName() == "" {
		vz.SetName("verrazzano")
	}

	return vz, nil
}

// MergeSetFlags merges yaml representing a set flag passed on the command line with a
// verrazano install resource.  A merged verrazzano install resource is returned.
func MergeSetFlags(gv schema.GroupVersion, vz clipkg.Object, overlayYAML string) (clipkg.Object, error) {
	baseYAML, err := yaml.Marshal(vz)
	if err != nil {
		return vz, err
	}
	vzYAML, err := overlayVerrazzano(gv, string(baseYAML), overlayYAML)
	if err != nil {
		return vz, err
	}

	obj := &unstructured.Unstructured{}
	err = yaml.Unmarshal([]byte(vzYAML), obj)
	if err != nil {
		return obj, fmt.Errorf("Failed to create a verrazzano install resource: %s", err.Error())
	}
	return obj, nil
}

// overlayVerrazzano overlays over base using JSON strategic merge.
func overlayVerrazzano(gv schema.GroupVersion, baseYAML string, overlayYAML string) (string, error) {
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
	mergedJSON, err := strategicpatch.StrategicMergePatch(baseJSON, overlayJSON, helpers.NewVerrazzanoForVersion(gv)())
	if err != nil {
		return "", fmt.Errorf("Failed to merge yaml: %s\n base object:\n%s\n override object:\n%s", err.Error(), baseJSON, overlayJSON)
	}

	mergedYAML, err := yaml.JSONToYAML(mergedJSON)
	if err != nil {
		return "", fmt.Errorf("Failed to create a verrazzano install resource: %s\n%s", err.Error(), mergedJSON)
	}

	return string(mergedYAML), nil
}

func checkGroupVersion(readBytes []byte, gv *schema.GroupVersion) error {
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(readBytes, obj); err != nil {
		// allow merge of unknown objects
		return nil
	}
	ogv := obj.GroupVersionKind().GroupVersion()
	if gv.Version == "" {
		*gv = ogv
	} else {
		if ogv != *gv {
			return fmt.Errorf("cannot merge objects with different group versions: %v != %v", *gv, ogv)
		}
	}
	return nil
}
