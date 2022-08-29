// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const servicePortsYaml = `- name: testPort
  port: 8080
  protocol: TCP
  targetPort: 2000
`
const lbGatewayYaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true
        k8s:
          replicaCount: 1
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          replicaCount: 1
          service:
            type: LoadBalancer
            ports:
            - name: testPort
              port: 8080
              protocol: TCP
              targetPort: 2000
            
`
const nodePortGatewayYaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true
        k8s:
          replicaCount: 1
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          replicaCount: 1
          service:
            type: NodePort
            ports:
            - name: testPort
              port: 8080
              protocol: TCP
              targetPort: 2000
            
`

func TestConfigureIngressGateway(t *testing.T) {
	var tests = []struct {
		name           string
		istioComponent *IstioComponent
		expectedData   istioTemplateData
	}{
		{
			name: "configure empty ingress gateway",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{},
			},
			expectedData: istioTemplateData{},
		},
		{
			name: "configure ingress gateway with kubernetes replicas",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
				},
			},
			expectedData: istioTemplateData{
				IngressKubernetes:   true,
				IngressReplicaCount: 1,
			},
		},
		{
			name: "configure ingress gateway of type node port having customand service ports",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
					Type: "NodePort",
					Ports: []v1.ServicePort{
						{
							Name: "testPort",
							Port: 8080,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2000,
							},
							Protocol: "TCP",
						},
					},
				},
			},
			expectedData: istioTemplateData{
				IngressKubernetes:   true,
				IngressReplicaCount: 1,
				IngressServiceType:  "NodePort",
				IngressServicePorts: servicePortsYaml,
			},
		},
		{
			name: "egress gateway with kubernetes replicas is not processed",
			istioComponent: &IstioComponent{
				Egress: &IstioEgressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
				},
			},
			expectedData: istioTemplateData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualTemplateData istioTemplateData
			err := configureIngressGateway(tt.istioComponent, &actualTemplateData)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedData, actualTemplateData)
		})
	}
}

func TestConfigureEgressGateway(t *testing.T) {
	var tests = []struct {
		name           string
		istioComponent *IstioComponent
		expectedData   istioTemplateData
	}{

		{
			name: "configure empty egress gateway",
			istioComponent: &IstioComponent{
				Egress: &IstioEgressSection{},
			},
			expectedData: istioTemplateData{},
		},
		{
			name: "configure egress gateway with kubernetes replicas",
			istioComponent: &IstioComponent{
				Egress: &IstioEgressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
				},
			},
			expectedData: istioTemplateData{
				EgressKubernetes:   true,
				EgressReplicaCount: 1,
			},
		},
		{
			name: "ingress gateway with kubernetes replicas, service ports is not processed",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
					Type: "NodePort",
					Ports: []v1.ServicePort{
						{
							Name: "testPort",
							Port: 8080,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2000,
							},
							Protocol: "TCP",
						},
					},
				},
			},
			expectedData: istioTemplateData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualTemplateData istioTemplateData
			err := configureEgressGateway(tt.istioComponent, &actualTemplateData)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedData, actualTemplateData)
		})
	}
}

func TestConfigureGateways(t *testing.T) {
	var tests = []struct {
		name           string
		istioComponent *IstioComponent
		externalIP     string
		expectedData   string
		err            error
	}{
		{
			name: "loadbalancer ingress gateway with kubernetes replicas, service ports and egress gateway with replicas",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
					Ports: []v1.ServicePort{
						{
							Name: "testPort",
							Port: 8080,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2000,
							},
							Protocol: "TCP",
						},
					},
				},
				Egress: &IstioEgressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec{
							Replicas: 1,
						},
					},
				},
			},
			expectedData: lbGatewayYaml,
		},
		{
			name: "nodeport ingress gateway with kubernetes replicas, service ports and egress gateway with replicas",
			istioComponent: &IstioComponent{
				Ingress: &IstioIngressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec: CommonKubernetesSpec{
							Replicas: 1,
						},
					},
					Type: "NodePort",
					Ports: []v1.ServicePort{
						{
							Name: "testPort",
							Port: 8080,
							TargetPort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2000,
							},
							Protocol: "TCP",
						},
					},
				},
				Egress: &IstioEgressSection{
					Kubernetes: &IstioKubernetesSection{
						CommonKubernetesSpec{
							Replicas: 1,
						},
					},
				},
			},
			expectedData: nodePortGatewayYaml,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualData, err := configureGateways(tt.istioComponent, tt.externalIP)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedData, actualData)
		})
	}
}
