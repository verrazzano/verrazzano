// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const argShape = `gateways.istio-ingressgateway.serviceAnnotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"`

// Specify the install args
var cr1 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  argShape,
			Value: "10Mbps",
		},
		{
			Name:  "global.defaultPodDisruptionBudget.enabled",
			Value: "false",
		},
		{
			Name:  "pilot.resources.requests.memory",
			Value: "128Mi",
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
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-egressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 1
      name: istio-egressgateway
    ingressGateways:
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-ingressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 1
        service:
          externalIPs:
          - 1.2.3.4
      name: istio-ingressgateway
  values:
    gateways:
      istio-ingressgateway:
        serviceAnnotations:
          service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
    global:
      defaultPodDisruptionBudget:
        enabled: false
    pilot:
      resources:
        requests:
          memory: 128Mi
`

// Specify the install args
var cr2 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  argShape,
			Value: "10Mbps",
		},
		{
			Name:  "global.defaultPodDisruptionBudget.enabled",
			Value: "false",
		},
		{
			Name:  "pilot.resources.requests.memory",
			Value: "128Mi",
		},
		{
			Name:  ExternalIPArg,
			Value: "1.2.3.4",
		},
	},
}

// Resulting YAML after the merge
const cr2Yaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-egressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 2
      name: istio-egressgateway
    ingressGateways:
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-ingressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 2
        service:
          externalIPs:
          - 1.2.3.4
      name: istio-ingressgateway
  values:
    gateways:
      istio-ingressgateway:
        serviceAnnotations:
          service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
    global:
      defaultPodDisruptionBudget:
        enabled: false
    pilot:
      resources:
        requests:
          memory: 128Mi
`

// Specify the install args
var cr3 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  argShape,
			Value: "10Mbps",
		},
		{
			Name:  "global.defaultPodDisruptionBudget.enabled",
			Value: "false",
		},
	},
}

// Resulting YAML after the merge
const cr3Yaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-egressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 1
      name: istio-egressgateway
    ingressGateways:
    - enabled: true
      k8s:
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                  - key: app
                    operator: In
                    values:
                    - istio-ingressgateway
                topologyKey: kubernetes.io/hostname
              weight: 100
        replicaCount: 1
      name: istio-ingressgateway
  values:
    gateways:
      istio-ingressgateway:
        serviceAnnotations:
          service.beta.kubernetes.io/oci-load-balancer-shape: 10Mbps
    global:
      defaultPodDisruptionBudget:
        enabled: false
`

// TestBuildIstioOperatorYaml tests the BuildIstioOperatorYaml function
// GIVEN an Verrazzano CR Istio component
// WHEN BuildIstioOperatorYaml is called
// THEN ensure that the result is correct.
func TestBuildIstioOperatorYaml(t *testing.T) {
	const indent = 2

	tests := []struct {
		testName string
		profile  vzapi.ProfileType
		value    *vzapi.IstioComponent
		expected string
	}{
		{
			testName: "Default Dev Profile Install",
			profile:  vzapi.Dev,
			value:    &cr1,
			expected: cr1Yaml,
		},
		{
			testName: "Default Prod Profile Install",
			profile:  vzapi.Prod,
			value:    &cr2,
			expected: cr2Yaml,
		},
		{
			testName: "Default Managed Cluster Install",
			profile:  vzapi.ManagedCluster,
			value:    &cr3,
			expected: cr3Yaml,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := BuildIstioOperatorYaml(test.value, test.profile)
			assert.NoError(err, s, "error merging yamls")
			assert.YAMLEq(test.expected, s, "Result does not match expected value")
		})
	}
}
