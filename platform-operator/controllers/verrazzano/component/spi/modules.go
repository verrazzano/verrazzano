// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type moduleSpecValue struct {
	VerrazzanoSpec interface{} `json:"verrazzanoSpec"`
}

// NewModuleSpecValue Creates the embedded JSON snippet to pass into a Module instance as a well-known Helm value
func NewModuleSpecValue(specValue interface{}) (*apiextensionsv1.JSON, error) {
	modSpec := &moduleSpecValue{
		VerrazzanoSpec: specValue,
	}
	var jsonBytes []byte
	var err error
	if jsonBytes, err = json.Marshal(modSpec); err != nil {
		return nil, err
	}
	return &apiextensionsv1.JSON{
		Raw: jsonBytes,
	}, nil
}
