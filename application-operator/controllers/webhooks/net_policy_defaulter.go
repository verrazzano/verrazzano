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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var logger = ctrl.Log.WithName("webhooks.network.policy")

type NetPolicyDefaulter struct {
	Client client.Client
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

		// create/update the network policy
		logger.Info("Ensuring Istio network policy exists", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newNetworkPolicy(appConfig)

		_, err := controllerutil.CreateOrUpdate(context.TODO(), n.Client, &netpol, func() error {
			netpol.Spec = newNetworkPolicySpec(appConfig.Namespace)
			return nil
		})

		if err != nil {
			logger.Error(err, "Unable to create or update network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
			return err
		}
	}

	return nil
}

// Cleanup deletes the Istio network policy associated with the app config.
func (n *NetPolicyDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	if !dryRun {
		// delete the network policy
		logger.Info("Deleting Istio network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newNetworkPolicy(appConfig)
		err := n.Client.Delete(context.TODO(), &netpol, &client.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			// log the error but don't return it so we continue processing
			logger.Error(err, "Unable to delete network policy, ignoring")
		}
	}

	return nil
}

// ensureNamespaceLabel fetches the namespace and checks to see if it has the Verrazzano
// namespace label. If not, we add the label.
func (n *NetPolicyDefaulter) ensureNamespaceLabel(namespace string) error {
	var ns corev1.Namespace
	namespacedName := types.NamespacedName{Name: namespace}
	err := n.Client.Get(context.TODO(), namespacedName, &ns)
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

		err = n.Client.Update(context.TODO(), &ns, &client.UpdateOptions{})
		if err != nil {
			logger.Error(err, "Unable to add label to namespace", "namespace", namespace)
			return err
		}
	}

	return nil
}

// newNetworkPolicy returns a NetworkPolicy struct populated with the namespace and name.
func newNetworkPolicy(appConfig *oamv1.ApplicationConfiguration) netv1.NetworkPolicy {
	return netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.IstioSystemNamespace,
			Name:      appConfig.Namespace + "-" + appConfig.Name,
		},
	}
}

// newNetworkPolicySpec returns a NetworkPolicySpec struct populated with the istiod
// ingress rule that allows ingress to istiod from the application namespace.
func newNetworkPolicySpec(namespace string) netv1.NetworkPolicySpec {
	const istiodTLSPort = 15012

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
