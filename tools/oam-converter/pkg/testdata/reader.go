// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testdata

import (
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
)

func ReadFromYAMLTemplate(template string) (map[string]interface{}, error) {
	yamlData, err := readTemplate(template)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = yaml.Unmarshal(yamlData, &data)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	return data, nil
}

func readTemplate(template string) ([]byte, error) {
	bytes, err := os.ReadFile("../../" + template)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
