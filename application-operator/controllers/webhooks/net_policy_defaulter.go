// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type NetPolicyDefaulter struct {
	Client          client.Client
	NamespaceClient typedv1.NamespaceInterface
}

// Default handles creating the Istio network policy for the application. The network policy is needed
// to allow ingress to the istiod pod from the Istio proxy sidecar that runs in the application pods.
// This function also adds the Verrazzano namespace label to the app config namespace (if needed) so
// the network policy can match the namespace with a selector.
func (n *NetPolicyDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	if appConfig.DeletionTimestamp != nil {
		log.Debug("App config is being deleted, nothing to do", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		return nil
	}

	if !dryRun {
		// label the namespace - retry if we get update conflicts
		for retryCount := 0; retryCount < 5; retryCount++ {
			err := n.ensureNamespaceLabel(appConfig.Namespace, log)
			if err == nil {
				break
			}

			if k8serrors.IsConflict(err) {
				continue
			}
			return err
		}

		// create/update the network policy
		log.Debugw("Ensuring Istio network policy exists", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newNetworkPolicy(appConfig)

		_, err := controllerutil.CreateOrUpdate(context.TODO(), n.Client, &netpol, func() error {
			netpol.Spec = newNetworkPolicySpec(appConfig.Namespace)
			return nil
		})

		if err != nil {
			log.Errorw(fmt.Sprintf("Failed to create or update network policy: %v", err), "namespace", appConfig.Namespace, "app config name", appConfig.Name)
			return err
		}
	}

	return nil
}

// Cleanup deletes the Istio network policy associated with the app config.
func (n *NetPolicyDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	if !dryRun {
		// delete the network policy
		log.Debugw("Deleting Istio network policy", "namespace", appConfig.Namespace, "app config name", appConfig.Name)
		netpol := newNetworkPolicy(appConfig)
		err := n.Client.Delete(context.TODO(), &netpol, &client.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			// log the error but don't return it so we continue processing
			log.Errorf("Failed to delete network policy, ignoring: %v", err)
		}
	}

	return nil
}

// ensureNamespaceLabel fetches the namespace and checks to see if it has the Verrazzano
// namespace label. If not, we add the label.
func (n *NetPolicyDefaulter) ensureNamespaceLabel(namespace string, log *zap.SugaredLogger) error {
	ns, err := n.NamespaceClient.Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Failed to fetch namespace %s: %v", namespace, err)
		return err
	}

	if ns.ObjectMeta.Labels == nil {
		ns.ObjectMeta.Labels = make(map[string]string)
	}

	val, exists := ns.ObjectMeta.Labels[constants.LabelVerrazzanoNamespace]
	if !exists || val != namespace {
		log.Debugw("Updating namespace with Verrazzano namespace label", "namespace", namespace)
		ns.ObjectMeta.Labels[constants.LabelVerrazzanoNamespace] = namespace

		_, err = n.NamespaceClient.Update(context.TODO(), ns, metav1.UpdateOptions{})
		if err != nil {
			_, err = vzlog.IgnoreConflictWithLog(fmt.Sprintf("Failed to add label to namespace %s: %v", namespace, err), err, log)
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
