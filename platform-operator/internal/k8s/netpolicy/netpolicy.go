// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpolicy

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	kubeSystemNamespace   = "kube-system"
	nginxIngressNamespace = "ingress-nginx"

	networkPolicyAPIVersion  = "networking.k8s.io/v1"
	networkPolicyKind        = "NetworkPolicy"
	networkPolicyPodName     = "verrazzano-platform-operator"
	networkPolicyPodName2    = "verrazzano-platform-operator-webhook"
	podAppLabel              = "app"
	verrazzanoNamespaceLabel = "verrazzano.io/namespace"
	k8sAppLabel              = "k8s-app"
	kubeDNSPodName           = "kube-dns"
	nginxControllerPodName   = "ingress-controller"
	appInstanceLabel         = "app.kubernetes.io/instance"
	appNameLabel             = "app.kubernetes.io/name"
	apiServerEndpointName    = "kubernetes"
)

// CreateOrUpdateNetworkPolicies creates or updates network policies for the platform operator to
// limit network ingress and egress.
func CreateOrUpdateNetworkPolicies(clientset kubernetes.Interface, client client.Client) ([]controllerutil.OperationResult, []error) {
	ip, port, err := getAPIServerIPAndPort(clientset)
	var opResults []controllerutil.OperationResult
	var errors []error
	if err != nil {
		opResults = append(opResults, controllerutil.OperationResultNone)
		errors = append(errors, err)
		return opResults, errors
	}

	netPolicies := newNetworkPolicies(ip, port)
	for _, netPolicy := range netPolicies {
		objKey := &netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: netPolicy.ObjectMeta.Name, Namespace: netPolicy.ObjectMeta.Namespace}}

		opResult, err := controllerutil.CreateOrUpdate(context.TODO(), client, objKey, func() error {
			netPolicy.Spec.DeepCopyInto(&objKey.Spec)
			return nil
		})
		opResults = append(opResults, opResult)
		errors = append(errors, err)

	}

	return opResults, errors
}

// getAPIServerIPAndPort returns the IP address and port of the Kubernetes API server.
func getAPIServerIPAndPort(clientset kubernetes.Interface) (string, int32, error) {
	endpoints, err := clientset.CoreV1().Endpoints(corev1.NamespaceDefault).Get(context.TODO(), apiServerEndpointName, metav1.GetOptions{})
	if err != nil {
		return "", 0, err
	}

	if len(endpoints.Subsets) > 0 && len(endpoints.Subsets[0].Addresses) > 0 && len(endpoints.Subsets[0].Ports) > 0 {
		return endpoints.Subsets[0].Addresses[0].IP, endpoints.Subsets[0].Ports[0].Port, nil
	}

	return "", 0, fmt.Errorf("unable to find a host and port for the kubernetes API server")
}

// newNetworkPolicy returns a populated NetworkPolicy with ingress and egress rules for this operator.
func newNetworkPolicies(apiServerIP string, apiServerPort int32) []*netv1.NetworkPolicy {
	tcpProtocol := corev1.ProtocolTCP
	udpProtocol := corev1.ProtocolUDP
	dnsPort := intstr.FromInt(53)
	httpsPort := intstr.FromInt(443)
	webhookPort := intstr.FromInt(9443)
	metricsPort := intstr.FromInt(9100)
	apiPort := intstr.FromInt(int(apiServerPort))
	apiServerCidr := apiServerIP + "/32"
	vponetpol := &netv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkPolicyAPIVersion,
			Kind:       networkPolicyKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoInstallNamespace,
			Name:      networkPolicyPodName,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					podAppLabel: networkPolicyPodName,
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeEgress,
				netv1.PolicyTypeIngress,
			},
			Egress: []netv1.NetworkPolicyEgressRule{
				{
					// egress for DNS
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &dnsPort,
						},
						{
							Protocol: &udpProtocol,
							Port:     &dnsPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: kubeSystemNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									k8sAppLabel: kubeDNSPodName,
								},
							},
						},
					},
				},
				{
					// egress to the kubernetes API server
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &apiPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							IPBlock: &netv1.IPBlock{
								CIDR: apiServerCidr,
							},
						},
					},
				},
				{
					// egress to the Nginx ingress controller (so we can register the cluster with Rancher)
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &httpsPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: nginxIngressNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appInstanceLabel: nginxControllerPodName,
								},
							},
						},
					},
				},
				{
					// egress to the webhooks
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &webhookPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: constants.VerrazzanoInstallNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appInstanceLabel: networkPolicyPodName2,
								},
							},
						},
					},
				},
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					From: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: constants.VerrazzanoMonitoringNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appNameLabel: constants.PrometheusStorageLabelValue,
								},
							},
						},
					},
					// ingress from Prometheus server for scraping metrics
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &metricsPort,
						},
					},
				},
			},
		},
	}
	webhooknetpol := &netv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkPolicyAPIVersion,
			Kind:       networkPolicyKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoInstallNamespace,
			Name:      networkPolicyPodName2,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					podAppLabel: networkPolicyPodName2,
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeEgress,
				netv1.PolicyTypeIngress,
			},
			Egress: []netv1.NetworkPolicyEgressRule{
				{
					// egress for DNS
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &dnsPort,
						},
						{
							Protocol: &udpProtocol,
							Port:     &dnsPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: kubeSystemNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									k8sAppLabel: kubeDNSPodName,
								},
							},
						},
					},
				},
				{
					// egress to the kubernetes API server
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &apiPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							IPBlock: &netv1.IPBlock{
								CIDR: apiServerCidr,
							},
						},
					},
				},
				{
					// egress to the Nginx ingress controller (so we can register the cluster with Rancher)
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &httpsPort,
						},
					},
					To: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: nginxIngressNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appInstanceLabel: nginxControllerPodName,
								},
							},
						},
					},
				},
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					// ingress from kubernetes API server for webhooks
					Ports: []netv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port:     &webhookPort,
						},
					},
				},
				{
					From: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: constants.VerrazzanoMonitoringNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appNameLabel: constants.PrometheusStorageLabelValue,
								},
							},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: constants.VerrazzanoInstallNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appNameLabel: networkPolicyPodName,
								},
							},
						},
					},
				},
			},
		},
	}
	netpols := []*netv1.NetworkPolicy{vponetpol, webhooknetpol}
	return netpols
}
