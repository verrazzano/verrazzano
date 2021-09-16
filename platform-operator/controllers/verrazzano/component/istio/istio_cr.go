// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)


// Build the IstioOperator CR YAML that will be passed as an override to istioctl
func buildOverrideCR(comp *vzapi.IstioComponent) (string, error) {
	// Build a list of YAML strings, one for each arg
	var yamls []string
	for _, arg := range comp.IstioInstallArgs {
		values := arg.ValueList
		if len(values) == 0 {
			values = []string{arg.Value}
		}
		yaml, err := vzyaml.Expand(0,arg.Name,values...)
		if err != nil {
			return "", err
		}
		yamls = append(yamls, yaml)
	}

	// Merge the YAML strings
	// For example,
	//  a:
	//    b: v1
	// plus
	//   a:
	//    c: v2
	// equals
	// a:
	//   b: v1
	//   c: v2


	return "", nil
}
