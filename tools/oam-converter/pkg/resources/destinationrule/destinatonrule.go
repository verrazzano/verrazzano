// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package destinationrule

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"google.golang.org/protobuf/types/known/durationpb"
	istionet "istio.io/api/networking/v1alpha3"
	istio "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// CreateDestinationFromRule creates a destination from the rule
func CreateDestinationFromRule(rule vzapi.IngressRule) (*istio.HTTPRouteDestination, error) {
	dest := &istio.HTTPRouteDestination{Destination: &istio.Destination{Host: rule.Destination.Host}}
	if rule.Destination.Port != 0 {
		dest.Destination.Port = &istio.PortSelector{Number: rule.Destination.Port}
	}
	return dest, nil

}

// CreateDestinationRule  creates or updates the DestinationRule.
func CreateDestinationRule(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string) (*istioclient.DestinationRule, error) {
	if rule.Destination.HTTPCookie != nil {
		destinationRule := &istioclient.DestinationRule{
			TypeMeta: metav1.TypeMeta{
				APIVersion: consts.DestinationRuleAPIVersion,
				Kind:       "DestinationRule"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: trait.Namespace,
				Name:      name},
		}
		namespace := &corev1.Namespace{}
		return mutateDestinationRule(destinationRule, rule, namespace)

	}
	return nil, nil
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRule(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, namespace *corev1.Namespace) (*istioclient.DestinationRule, error) {

	dest, err := CreateDestinationFromRule(rule)

	if err != nil {
		print(err)
		return nil, err
	}

	mode := istionet.ClientTLSSettings_DISABLE
	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		mode = istionet.ClientTLSSettings_ISTIO_MUTUAL
	}
	destinationRule.Spec = istionet.DestinationRule{
		Host: dest.Destination.Host,
		TrafficPolicy: &istionet.TrafficPolicy{
			Tls: &istionet.ClientTLSSettings{
				Mode: mode,
			},
			LoadBalancer: &istionet.LoadBalancerSettings{
				LbPolicy: &istionet.LoadBalancerSettings_ConsistentHash{
					ConsistentHash: &istionet.LoadBalancerSettings_ConsistentHashLB{
						HashKey: &istionet.LoadBalancerSettings_ConsistentHashLB_HttpCookie{
							HttpCookie: &istionet.LoadBalancerSettings_ConsistentHashLB_HTTPCookie{
								Name: rule.Destination.HTTPCookie.Name,
								Path: rule.Destination.HTTPCookie.Path,
								Ttl:  durationpb.New(rule.Destination.HTTPCookie.TTL * time.Second)},
						},
					},
				},
			},
		},
	}

	return destinationRule, nil
}
