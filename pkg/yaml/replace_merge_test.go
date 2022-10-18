// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// rmBase is the base of a nested merge with a list
const rmBase = `
name: base
host:
  ip: 1.2.3.4
  name: foo
platform:
  vendor: company1
  os:
    name: linux
    patches:
    - version: 0.5.0
      date: 01/01/2020
`

// rmOverlay is the overlay of a nested merge with a list
const rmOverlay = `
host:
  name: bar
platform:
  os:
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

// rmMerged is the result of a nested merge where the list is replaced
const rmMerged = `
name: base
host:
  ip: 1.2.3.4
  name: bar
platform:
  vendor: company1
  os:
    name: linux
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

// TestMergeReplace tests the ReplacementMerge function with nested YAML
// GIVEN a set of nested YAML strings with embedded lists
// WHEN ReplacementMerge is called
// THEN ensure that the merged result is correct.
func TestMergeReplace(t *testing.T) {
	assert := assert.New(t)
	merged, err := ReplacementMerge(rmBase, rmOverlay)
	assert.NoError(err, merged, "error merging nested yaml")
	assert.YAMLEq(rmMerged, merged, "nested yaml should be equal")
}

// istioBase is the base of an IstioOperator YAML
const istioBase = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true

  # Global values passed through to helm global.yaml.
  # Please keep this in sync with manifests/charts/global.yaml
  values:
    global:
      gateways:
        istio-ingressgateway:
          serviceAnnotations:
            service.beta.kubernetes.io/oci-load-balancer-shape: flexible
`

// istiOverlay is the overlay of an IstioOperator YAML
const istiOverlay = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          service:
            type: ClusterIP
            externalIPs:
            - 1.2.3.4
`

// istioMerged is the result of a merge of IstioOperator YAMLs
const istioMerged = `
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
            service.beta.kubernetes.io/oci-load-balancer-shape: flexible
`

const jaegerBase = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    meshConfig:
      defaultConfig:
        tracing:
          zipkin:
            address: abc.xyz
          tlsSettings:
            mode: ISTIO_MUTUAL
`
const jaegerOverlay = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    meshConfig:
      defaultConfig:
        tracing:
          sampling: 90
          zipkin:
            address: cdef.xyz
`

const jaegerMerged = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
    meshConfig:
      defaultConfig:
        tracing:
          sampling: 90
          zipkin:
            address: cdef.xyz
          tlsSettings:
            mode: ISTIO_MUTUAL
`

// TestMergeReplaceIstio tests the ReplacementMerge function with IstiOperator YAML
// GIVEN a set of nested YAML strings with embedded lists
// WHEN ReplacementMerge is called
// THEN ensure that the merged result is correct.
func TestMergeReplaceIstio(t *testing.T) {
	assert := assert.New(t)
	merged, err := ReplacementMerge(istioBase, istiOverlay)
	assert.NoError(err, merged, "error merging Istio YAML")
	assert.YAMLEq(istioMerged, merged, "incorrect Istio merged YAML")
}

// TestMergeReplaceJaeger tests the ReplacementMerge function with IstiOperator YAML
// GIVEN a set of nested YAML strings with embedded lists
// WHEN ReplacementMerge is called
// THEN ensure that the merged result is correct.
func TestMergeReplaceJaeger(t *testing.T) {
	assert := assert.New(t)
	merged, err := ReplacementMerge(jaegerBase, jaegerOverlay)
	assert.NoError(err, merged, "error merging Istio YAML")
	assert.YAMLEq(jaegerMerged, merged, "incorrect Istio merged YAML")
}

// Complete replace YAML
const rm1 = `
k1: rm1-v1
k2:
  k3: rm1-k2.3
  k4: rm1-k2.4
`
const rm2 = `
k1: rm2-v1
k2:
  k3: rm2-k2.3
  k4: rm2-k2.4
`

// rm2 merged on top of rm1
const rm1_2 = `
k1: rm2-v1
k2:
  k3: rm2-k2.3
  k4: rm2-k2.4
`

// Partial replace YAML
const rm3 = `
k1: rm3-v1
k2:
  k4: rm3-k2.4
`
const rm4 = `
k1: rm4-v1
k2:
  k3: rm4-k2.3
`

// rm4 merged on top of rm3
const rm3_4 = `
k1: rm4-v1
k2:
  k3: rm4-k2.3
  k4: rm3-k2.4
`

// Appending keys YAML
const rm5 = `
k1: rm5-v1
`
const rm6 = `
k2:
  k3: rm6-k2.3
`
const rm7 = `
k2:
  k4: rm7-k2.4
`

// rm4 merged on top of rm3
const rm5_6_7 = `
k1: rm5-v1
k2:
  k3: rm6-k2.3
  k4: rm7-k2.4
`

// TestReplaceMany tests the ReplacementMerge function
// GIVEN a set of yamls
// WHEN ReplacementMerge is called
// THEN ensure that the result is correct.
func TestReplaceMany(t *testing.T) {
	tests := []struct {
		testName string
		values   []string
		expected string
	}{
		{
			testName: "1",
			values:   []string{rm1, rm2},
			expected: rm1_2,
		},
		{
			testName: "2",
			values:   []string{rm3, rm4},
			expected: rm3_4,
		},
		{
			testName: "3",
			values:   []string{rm5, rm6, rm7},
			expected: rm5_6_7,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := ReplacementMerge(test.values...)
			assert.NoError(err, s, "error merging yamls")
			assert.YAMLEq(test.expected, s, "Result does not match expected value")
		})
	}
}
