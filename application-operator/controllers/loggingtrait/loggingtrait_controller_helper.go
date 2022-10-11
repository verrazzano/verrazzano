// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"encoding/json"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// struct2Unmarshal - Struct to unstructured
func struct2Unmarshal(obj interface{}) (unstructured.Unstructured, error) {
	marshal, err := json.Marshal(obj)
	var c unstructured.Unstructured
	c.UnmarshalJSON(marshal)
	return c, err
}

// appendSliceOfInterface - Append two slices of interfaces in to one slice without duplicates
func appendSliceOfInterface(aSlice []interface{}, bSlice []interface{}) []interface{} {

	res := make([]interface{}, 0)
	res = append(res, bSlice...)

	for _, k := range aSlice {
		jndex := -1
		for j, l := range bSlice {
			if reflect.DeepEqual(k, l) {
				jndex = j
			}
		}
		if jndex == -1 {
			res = append(res, k)
		}
	}

	return res

}

// locateContainersField locate the containers field
func locateContainersField(res *unstructured.Unstructured) (bool, []string) {
	var containersFieldPath []string
	var ok = false
	kind := res.GetKind()

	switch kind {
	case "Pod":
		containersFieldPath = []string{"spec", "containers"}
		ok = true
	case "ContainerizedWorkload":
		containersFieldPath = []string{"spec", "containers"}
		ok = true
	case "Deployment":
		containersFieldPath = []string{"spec", "template", "spec", "containers"}
		ok = true
	case "StatefuleSet":
		containersFieldPath = []string{"spec", "template", "spec", "containers"}
		ok = true
	case "DaemonSet":
		containersFieldPath = []string{"spec", "template", "spec", "containers"}
		ok = true
	}

	return ok, containersFieldPath
}

// locateVolumesField locate the volumes field
func locateVolumesField(res *unstructured.Unstructured) (bool, []string) {
	var volumeFieldPath []string
	var ok = false
	kind := res.GetKind()

	switch kind {
	case "Pod":
		volumeFieldPath = []string{"spec", "volumes"}
		ok = true
	case "ContainerizedWorkload":
		volumeFieldPath = []string{"spec", "volumes"}
		ok = true
	case "Deployment":
		volumeFieldPath = []string{"spec", "template", "spec", "volumes"}
		ok = true
	case "StatefuleSet":
		volumeFieldPath = []string{"spec", "template", "spec", "volumes"}
		ok = true
	case "DaemonSet":
		volumeFieldPath = []string{"spec", "template", "spec", "volumes"}
		ok = true
	}

	return ok, volumeFieldPath
}
