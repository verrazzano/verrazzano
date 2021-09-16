// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const anno1 = `
gateways.istio-ingressgateway.serviceAnnotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"`

// Appending keys YAML
var cr1 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{{
		Name:  anno1,
		Value: "10Mbps",
	}},
}

// Resulting YAML after the merge
const cr1Yaml = `
gateways:
  istio-ingressgateway:
    serviceAnnotations:
      service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
`

// TestBuildOverrideCR tests the BuildOverrideCR function
// GIVEN an Verrazzano CR Istio component
// WHEN BuildOverrideCR is called
// THEN ensure that the result is correct.
func TestBuildOverrideCR(t *testing.T) {
	const indent = 2

	tests := []struct {
		testName string
		value    *vzapi.IstioComponent
		expected string
	}{
		{
			testName: "1",
			value:    &cr1,
			expected: cr1Yaml,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := BuildOverrideCR(test.value)
			assert.NoError(err, s, "error merging yamls")
			assert.YAMLEq(test.expected, s, "Result does not match expected value")
		})
	}
}
