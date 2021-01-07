// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// WriteYaml writes/marshalls the obj to a yaml file.
func WriteYaml(path string, obj interface{}) (string, error) {
	fileout, _ := filepath.Abs(path)
	bytes, err := ToYaml(obj)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(fileout, bytes, 0644)
	return fileout, err
}

// ToYaml marshalls the obj to a yaml
func ToYaml(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}
