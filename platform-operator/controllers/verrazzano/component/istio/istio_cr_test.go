// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const ComponentInstallArgShape = `gateways.istio-ingressgateway.serviceAnnotations."service\.beta\.kubernetes\.io/oci-load-balancer-shape"`

var testScheme = runtime.NewScheme()

func init() {
	_ = istioclisec.AddToScheme(testScheme)
	_ = corev1.AddToScheme(testScheme)
}

var enabled = true

var jaegerEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &enabled,
			},
		},
	},
}

var prodIstioIngress = &vzapi.IstioIngressSection{
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
	Type: vzapi.NodePort,
	Ports: []corev1.ServicePort{
		{
			Name:     "port1",
			Protocol: "TCP",
			Port:     8000,
			NodePort: 32443,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 2000,
			},
		},
	},
}

var prodIstioEgress = &vzapi.IstioEgressSection{
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
}

// Prod Profile defaults for replicas and affinity
// Extra Install Args
var cr1 = vzapi.IstioComponent{
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  ComponentInstallArgShape,
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
	Ingress: prodIstioIngress,
	Egress:  prodIstioEgress,
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
          type: NodePort
          ports:
          - name: port1
            protocol: TCP
            port: 8000
            nodePort: 32443
            targetPort: 2000
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
    meshConfig:
      defaultConfig:
        tracing:
          tlsSettings:
            mode: ISTIO_MUTUAL
          zipkin:
            address: jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411
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
			Name:  ComponentInstallArgShape,
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
          type: LoadBalancer
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
    meshConfig:
      defaultConfig:
        tracing:
          tlsSettings:
            mode: ISTIO_MUTUAL
          zipkin:
            address: jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411
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
    - enabled: true
      k8s:
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
        replicaCount: 3
      name: istio-egressgateway
    ingressGateways:
    - enabled: true
      k8s:
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
        replicaCount: 3
        service:
          type: LoadBalancer
      name: istio-ingressgateway
  values:
    meshConfig:
      defaultConfig:
        tracing:
          tlsSettings:
            mode: ISTIO_MUTUAL
          zipkin:
            address: jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411
`

var cr4 = &vzapi.IstioComponent{
	Enabled: &enabled,
	Ingress: prodIstioIngress,
	Egress:  prodIstioEgress,
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  "meshConfig.enableTracing",
			Value: "true",
		},
		{
			Name:  "meshConfig.defaultConfig.tracing.sampling",
			Value: "100",
		},
	},
}

var cr5 = &vzapi.IstioComponent{
	Enabled: &enabled,
	Ingress: prodIstioIngress,
	Egress:  prodIstioEgress,
	IstioInstallArgs: []vzapi.InstallArgs{
		{
			Name:  "meshConfig.enableTracing",
			Value: "true",
		},
		{
			Name:  "meshConfig.defaultConfig.tracing.sampling",
			Value: "100",
		},
		{
			Name:  "meshConfig.defaultConfig.tracing.zipkin.address",
			Value: "jaeger-collector.foo.svc.cluster.local:5555",
		},
	},
}

var cr4Yaml = `
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
          ports:
          - name: port1
            nodePort: 32443
            port: 8000
            protocol: TCP
            targetPort: 2000
          type: NodePort
      name: istio-ingressgateway
  values:
    meshConfig:
      defaultConfig:
        tracing:
          sampling: 100
          tlsSettings:
            mode: ISTIO_MUTUAL
          zipkin:
            address: jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411
      enableTracing: true
`

var cr5Yaml = `
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
          ports:
          - name: port1
            nodePort: 32443
            port: 8000
            protocol: TCP
            targetPort: 2000
          type: NodePort
      name: istio-ingressgateway
  values:
    meshConfig:
      defaultConfig:
        tracing:
          sampling: 100
          tlsSettings:
            mode: ISTIO_MUTUAL
          zipkin:
            address: jaeger-collector.foo.svc.cluster.local:5555
      enableTracing: true
`

// TestBuildIstioOperatorYaml tests the BuildIstioOperatorYaml function
// GIVEN an Verrazzano CR Istio component
// WHEN BuildIstioOperatorYaml is called
// THEN ensure that the result is correct.
func TestBuildIstioOperatorYaml(t *testing.T) {
	fakeCtx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vzapi.Verrazzano{}, false)
	collectorLabels := map[string]string{
		constants.KubernetesAppLabel: constants.JaegerCollectorService,
	}
	clientForJaeger := fake.NewClientBuilder().WithScheme(testScheme).
		WithObjects(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testZipkinNamespace,
				Name:      "jaeger-collector-headless",
				Labels:    collectorLabels,
			},
		},
			&testZipkinService).Build()
	tests := []struct {
		testName string
		value    *vzapi.IstioComponent
		expected string
		ctx      spi.ComponentContext
	}{
		{
			testName: "Default Prod Profile Install",
			value:    &cr1,
			expected: cr1Yaml,
			ctx:      fakeCtx,
		},
		{
			testName: "Default Dev and Managed-Cluster Profile Install",
			value:    &cr2,
			expected: cr2Yaml,
			ctx:      fakeCtx,
		},
		{
			testName: "Override Affinity and Replica",
			value:    &cr3,
			expected: cr3Yaml,
			ctx:      fakeCtx,
		},
		{
			testName: "When Jaeger Operator is enabled, without install args override default tracing URL is used",
			value:    cr4,
			expected: cr4Yaml,
			ctx:      spi.NewFakeContext(clientForJaeger, jaegerEnabledCR, false),
		},
		{
			testName: "When Jaeger Operator is enabled, with install args override, user provided tracing URL is used",
			value:    cr5,
			expected: cr5Yaml,
			ctx:      spi.NewFakeContext(clientForJaeger, jaegerEnabledCR, false),
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			a := assert.New(t)
			s, err := BuildIstioOperatorYaml(test.ctx, test.value)
			fmt.Println(s)
			a.NoError(err, s, "error merging yamls")
			a.YAMLEq(test.expected, s, "Result does not match expected value")
		})
	}
}
