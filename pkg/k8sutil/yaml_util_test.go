// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMarshal(t *testing.T) {
	obj1 := makeUnstructured("obj1")
	obj2 := makeUnstructured("obj2")
	objs := []unstructured.Unstructured{
		{Object: obj1},
		{Object: obj2},
	}
	obj1Bytes, err := yaml.Marshal(obj1)
	assert.NoError(t, err)
	obj2Bytes, err := yaml.Marshal(obj2)
	assert.NoError(t, err)
	marshalled, err := Marshal(objs)
	assert.NoError(t, err)
	marshalledStr := string(marshalled)
	fmt.Println(marshalledStr)
	assert.Contains(t, marshalledStr, fmt.Sprintf("%s\n%s\n%s", string(obj1Bytes), sep, string(obj2Bytes)))
}

func makeUnstructured(objname string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "myapi/v1",
		"kind":       "SomeObject",
		"metadata": map[string]interface{}{
			"name":      objname,
			"namespace": "ns1",
		},
	}
}
