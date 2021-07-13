// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const istiodTLSPort = 15012

var logger = ctrl.Log.WithName("webhooks.network.policy")

type NetPolicyDefaulter struct {
	Client          client.Client
	NamespaceClient typedv1.NamespaceInterface
}

// Default handles creating the Istio network policy for the application. The network policy is needed
// to allow ingress to the istiod pod from the Istio proxy sidecar that runs in the application pods.
// This function also adds the Verrazzano namespace label to the app config namespace (if needed) so
// the network policy can match the namespace with a selector.
func (n *NetPolicyDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	if appConfig.DeletionTimestamp != nil {
		logger.Info("App config is being deleted, nothing to do", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		return nil
	}

	if !dryRun {
		// label the namespace - retry if we get update conflicts
		for retryCount := 0; retryCount < 5; retryCount++ {
			err := n.ensureNamespaceLabel(appConfig.Namespace)
			if err == nil {
				break
			}

			if k8serrors.IsConflict(err) {
				continue
			}
			return err
		}

		// create/update the istiod network policy
		logger.Info("Ensuring Istio network policy exists", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newIstiodNetworkPolicy(appConfig)

		_, err := controllerutil.CreateOrUpdate(context.TODO(), n.Client, &netpol, func() error {
			netpol.Spec = newIstiodNetworkPolicySpec(appConfig.Namespace)
			return nil
		})

		if err != nil {
			logger.Error(err, "Unable to create or update istiod network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
			return err
		}

		// create/update the app default network policy
		logger.Info("Ensuring app default network policy exists", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol = newAppDefaultNetworkPolicy(appConfig)

		_, err = controllerutil.CreateOrUpdate(context.TODO(), n.Client, &netpol, func() error {
			netpol.Spec = newAppDefaultNetworkPolicySpec(appConfig)
			return nil
		})

		if err != nil {
			logger.Error(err, "Unable to create or update app default network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
			return err
		}
	}

	return nil
}

// Cleanup deletes the Istio network policy associated with the app config.
func (n *NetPolicyDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	if !dryRun {
		// delete the istiod network policy
		logger.Info("Deleting Istiod network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newIstiodNetworkPolicy(appConfig)
		err := n.Client.Delete(context.TODO(), &netpol, &client.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			// log the error but don't return it so we continue processing
			logger.Error(err, "Unable to delete istiod network policy, ignoring")
		}
		// delete the app default network policy
		logger.Info("Deleting app default network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol = newAppDefaultNetworkPolicy(appConfig)
		err = n.Client.Delete(context.TODO(), &netpol, &client.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			// log the error but don't return it so we continue processing
			logger.Error(err, "Unable to delete the app default network policy, ignoring")
		}
	}

	return nil
}

// ensureNamespaceLabel fetches the namespace and checks to see if it has the Verrazzano
// namespace label. If not, we add the label.
func (n *NetPolicyDefaulter) ensureNamespaceLabel(namespace string) error {
	ns, err := n.NamespaceClient.Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Unable to fetch namespace", "namespace", namespace)
		return err
	}

	if ns.ObjectMeta.Labels == nil {
		ns.ObjectMeta.Labels = make(map[string]string)
	}

	val, exists := ns.ObjectMeta.Labels[constants.LabelVerrazzanoNamespace]
	if !exists || val != namespace {
		logger.Info("Updating namespace with Verrazzano namespace label", "namespace", namespace)
		ns.ObjectMeta.Labels[constants.LabelVerrazzanoNamespace] = namespace

		_, err = n.NamespaceClient.Update(context.TODO(), ns, metav1.UpdateOptions{})
		if err != nil {
			logger.Error(err, "Unable to add label to namespace", "namespace", namespace)
			return err
		}
	}

	return nil
}

// newIstiodNetworkPolicy returns a NetworkPolicy for accessing istiod, populated with the istio namespace and policy name.
func newIstiodNetworkPolicy(appConfig *oamv1.ApplicationConfiguration) netv1.NetworkPolicy {
	return netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.IstioSystemNamespace,
			Name:      appConfig.Namespace + "-" + appConfig.Name,
		},
	}
}

// newIstiodNetworkPolicySpec returns a NetworkPolicySpec for accessing istiod, populated with the istiod
// ingress rule that allows ingress to istiod from the application namespace.
func newIstiodNetworkPolicySpec(namespace string) netv1.NetworkPolicySpec {
	tcpProtocol := corev1.ProtocolTCP
	istiodPort := intstr.FromInt(istiodTLSPort)

	return netv1.NetworkPolicySpec{
		PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
		},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				// ingress from app namespace to istiod
				Ports: []netv1.NetworkPolicyPort{
					{
						Protocol: &tcpProtocol,
						Port:     &istiodPort,
					},
				},
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: namespace},
						},
					},
				},
			},
		},
	}
}

// newAppDefaultNetworkPolicy returns a default NetworkPolicy for app, populated with the app namespace and policy name.
func newAppDefaultNetworkPolicy(appConfig *oamv1.ApplicationConfiguration) netv1.NetworkPolicy {
	return netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: appConfig.Namespace,
			Name:      appConfig.Namespace + "-" + appConfig.Name,
		},
	}
}

// newAppDefaultNetworkPolicySpec returns a default NetworkPolicySpec for app that
// Allow Istio Ingress Gateway access to the pods as specified in the IngressTrait.
// Allow all pods in the application ingress to all other pods in the app on all ports
// Allow Prometheus access to the metrics port.
// Allow Coherence operator access to the port for Coherence clusters
// Allow WebLogic operator access to all ports for WebLogic domains
// Allow egress to Istiod for Envoy sidecar
// Allow egress to the Istio egress gateway
func newAppDefaultNetworkPolicySpec(appConfig *oamv1.ApplicationConfiguration) netv1.NetworkPolicySpec {
	tcpProtocol := corev1.ProtocolTCP
	udpProtocol := corev1.ProtocolUDP
	istiodPort := intstr.FromInt(istiodTLSPort)
	dnsPort := intstr.FromInt(53)
	coherencePort := intstr.FromInt(8000)
	// TODO how to get app ports and metrics ports from app config

	return netv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{"app.oam.dev/name": appConfig.Name},
		},
		PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
			netv1.PolicyTypeEgress,
		},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				// all pods in the application to all other pods in the app on all ports
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: appConfig.Namespace},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app.oam.dev/name": appConfig.Name},
						},
					},
				},
			},
			{
				// Istio Ingress Gateway access to the pods
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "istio-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "istio-ingressgateway"},
						},
					},
				},
			},
			{
				// Prometheus access to the metrics port
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "verrazzano-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "system-prometheus"},
						},
					},
				},
			},
			{
				// Coherence operator access to ports for Coherence clusters
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "verrazzano-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "coherence-operator"},
						},
					},
				},
			},
			{
				// WebLogic operator access to ports for WebLogic domains
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "verrazzano-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "weblogic-operator"},
						},
					},
				},
			},
		},
		Egress: []netv1.NetworkPolicyEgressRule{
			{
				// all pods in the application to all other pods in the app on all ports
				To: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: appConfig.Namespace},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app.oam.dev/name": appConfig.Name},
						},
					},
				},
			},
			{
				// egress to istiod
				Ports: []netv1.NetworkPolicyPort{
					{
						Protocol: &tcpProtocol,
						Port:     &istiodPort,
					},
				},
				To: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "istio-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "istiod"},
						},
					},
				},
			},
			{
				// egress to Istio egress gateway
				To: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "istio-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "istio-egressgateway"},
						},
					},
				},
			},
			{
				// egress to coredns
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
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "kube-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"k8s-app": "kube-dns"},
						},
					},
				},
			},
			{
				// egress to coherence operator
				Ports: []netv1.NetworkPolicyPort{
					{
						Protocol: &tcpProtocol,
						Port:     &coherencePort,
					},
				},
				To: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.LabelVerrazzanoNamespace: "verrazzano-system"},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "coherence-operator"},
						},
					},
				},
			},
		},
	}
}
