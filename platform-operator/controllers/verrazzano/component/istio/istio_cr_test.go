// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const argShape = `
gateways.istio-ingressgateway.serviceAnnotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"`

// Specify the install args
var cr1 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  argShape,
			Value: "10Mbps",
		},
		{
			Name:  ExternalIPArg,
			Value: "1.2.3.4",
		},
	},
}

// Resulting YAML after the merge
const cr1Yaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          service:
            type: ClusterIP
            externalIPs:
            - 1.2.3.4

  # Global values passed through to helm global.yaml.
  # Please keep this in sync with manifests/charts/global.yaml
  values:
    global:
      gateways:
        istio-ingressgateway:
          serviceAnnotations:
            service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
`

// TestBuildIstioOperatorYaml tests the BuildIstioOperatorYaml function
// GIVEN an Verrazzano CR Istio component
// WHEN BuildIstioOperatorYaml is called
// THEN ensure that the result is correct.
func TestBuildIstioOperatorYaml(t *testing.T) {
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
			s, err := BuildIstioOperatorYaml(test.value)
			assert.NoError(err, s, "error merging yamls")
			assert.YAMLEq(test.expected, s, "Result does not match expected value")
		})
	}
}
