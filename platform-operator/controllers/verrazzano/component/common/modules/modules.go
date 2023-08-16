// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package modules

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

// NewModuleConfigHelmValuesWrapper Creates a JSON snippet to pass into a Module instance as a well-known Helm value
//
// Takes any object and marshals it into a Helm values hierarchy under "verrazzano.module.spec".  This allows conversion
// of any configuration in the Verrazzano CR needed by a module into a form consumable by that module.
//
// As a result only changes to the values in the passed-in structure should result in a reconile of the underlying
// Module instance.  Eventually this wrapper can be used to divorce the Module impls from the Verrazzano CR and
// Effective CR.
func NewModuleConfigHelmValuesWrapper(configObject interface{}) (*apiextensionsv1.JSON, error) {
	if configObject == nil {
		return &apiextensionsv1.JSON{}, nil
	}
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
