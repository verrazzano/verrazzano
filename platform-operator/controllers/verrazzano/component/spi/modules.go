// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type ModuleValuesSection struct {
	Spec interface{} `json:"spec,omitempty"`
}

type VerrazzanoValues struct {
	Module ModuleValuesSection `json:"module,omitempty"`
}

type VerrazzanoValuesConfig struct {
	Verrazzano VerrazzanoValues `json:"verrazzano"`
}

// NewModuleSpecValue Creates a JSON snippet to pass into a Module instance as a well-known Helm value
func NewModuleSpecValue(configObject interface{}) (*apiextensionsv1.JSON, error) {
	modSpec := &VerrazzanoValuesConfig{
		Verrazzano: VerrazzanoValues{
			Module: ModuleValuesSection{
				Spec: configObject,
			},
		},
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
