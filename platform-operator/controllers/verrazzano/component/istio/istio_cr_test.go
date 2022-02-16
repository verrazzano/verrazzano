// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const argShape = `gateways.istio-ingressgateway.serviceAnnotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"`

var enabled = true

// Prod Profile defaults for replicas and affinity
// Extra Install Args
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
	Enabled: &enabled,
	Ingress: &vzapi.IstioIngressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 2,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "In",
												Values: []string{
													"istio-ingressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	},
	Egress: &vzapi.IstioEgressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 2,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "In",
												Values: []string{
													"istio-egressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
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

// Dev/Managed Cluster Profile defaults for replicas and affinity
// Extra Install Args
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
	Enabled: &enabled,
	Ingress: &vzapi.IstioIngressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 1,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "In",
												Values: []string{
													"istio-ingressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	},
	Egress: &vzapi.IstioEgressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 1,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "In",
												Values: []string{
													"istio-egressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
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

// Override Affinity and Replicas
var cr3 = vzapi.IstioComponent{
	Enabled: &enabled,
	Ingress: &vzapi.IstioIngressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 3,
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: nil,
						},
						PreferredDuringSchedulingIgnoredDuringExecution: nil,
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution:  nil,
						PreferredDuringSchedulingIgnoredDuringExecution: nil,
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 30,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "NotIn",
												Values: []string{
													"istio-ingressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	},
	Egress: &vzapi.IstioEgressSection{
		Kubernetes: &vzapi.IstioKubernetesSection{
			CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
				Replicas: 3,
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: nil,
						},
						PreferredDuringSchedulingIgnoredDuringExecution: nil,
					},
					PodAffinity: &corev1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution:  nil,
						PreferredDuringSchedulingIgnoredDuringExecution: nil,
					},
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: nil,
						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
							{
								Weight: 30,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: nil,
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "NotIn",
												Values: []string{
													"istio-egressgateway",
												},
											},
										},
									},
									Namespaces:  nil,
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
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
      - name: istio-egressgateway
        enabled: true
        k8s:
          replicaCount: 3
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms: null
            podAffinity: {}
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchExpressions:
                    - key: app
                      operator: NotIn
                      values:
                      - istio-egressgateway
                  topologyKey: kubernetes.io/hostname
                weight: 30
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          replicaCount: 3
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms: null
            podAffinity: {}
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchExpressions:
                    - key: app
                      operator: NotIn
                      values:
                      - istio-ingressgateway
                  topologyKey: kubernetes.io/hostname
                weight: 30
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
			testName: "Default Prod Profile Install",
			value:    &cr1,
			expected: cr1Yaml,
		},
		{
			testName: "Default Dev and Managed-Cluster Profile Install",
			value:    &cr2,
			expected: cr2Yaml,
		},
		{
			testName: "Override Affinity and Replica",
			value:    &cr3,
			expected: cr3Yaml,
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
