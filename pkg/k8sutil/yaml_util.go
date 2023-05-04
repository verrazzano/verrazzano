// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Marshal a list of unstructured objects as YAML to a byte array
func Marshal(objs []unstructured.Unstructured) ([]byte, error) {
	buffer := bytes.Buffer{}
	sepLine := fmt.Sprintf("\n%s\n", sep)
	for _, obj := range objs {
		nextBytes, err := yaml.Marshal(obj.Object)
		if err != nil {
			return nil, err
		}
		buffer.Write(nextBytes)
		buffer.WriteString(sepLine)
	}
	return buffer.Bytes(), nil
}
