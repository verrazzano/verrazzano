// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"
)

//Struct to unstructured
func struct2Unmarshal(obj interface{}) (unstructured.Unstructured, error) {
	marshal, err := json.Marshal(obj)
	var c unstructured.Unstructured
	c.UnmarshalJSON(marshal)
	return c, err
}

func apiVersion2GroupVersion(str string) (string, string) {
	strs := strings.Split(str, "/")
	if len(strs) == 2 {
		return strs[0], strs[1]
	}
	// core type
	return "", strs[0]
}

//locateField of a given resource and try to see if it has fields of type array.
func locateField(document openapi.Resources, res *unstructured.Unstructured, fieldPaths [][]string) (bool, []string) {

	g, v := apiVersion2GroupVersion(res.GetAPIVersion())

	schema := document.LookupResource(schema.GroupVersionKind{
		Group:   g,
		Version: v,
		Kind:    res.GetKind(),
	})

	for _, containerFieldPath := range fieldPaths {
		field, err := explain.LookupSchemaForField(schema, containerFieldPath)
		if err == nil && field != nil {
			_, ok := field.(*proto.Array)
			return ok, containerFieldPath
		}
	}
	return false, nil
}

//locateContainersField locate the containers field
func locateContainersField(document openapi.Resources, res *unstructured.Unstructured) (bool, []string) {
	//This is the most common path to the containers field
	containersFieldPaths := [][]string{
		//This is the path to the containers field of the Pod resource
		{"spec", "containers"},
		//This is the path to the containers field of the Deployments,StatefulSet,ReplicaSet resource
		{"spec", "template", "spec", "containers"},
	}
	return locateField(document, res, containersFieldPaths)
}

//locateVolumesField locate the volumes field
func locateVolumesField(document openapi.Resources, res *unstructured.Unstructured) (bool, []string) {
	//This is the most common path to the volumes field
	volumesFieldPaths := [][]string{
		//This is the path to the volumes field of the Pod resource
		{"spec", "volumes"},
		//This is the path to the volumes field of the Deployments,StatefulSet,ReplicaSet resource
		{"spec", "template", "spec", "volumes"},
	}
	return locateField(document, res, volumesFieldPaths)
}