// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"errors"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationRule"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"google.golang.org/protobuf/types/known/durationpb"
	istionet "istio.io/api/networking/v1alpha3"
	istio "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// createOfUpdateDestinationRule creates or updates the DestinationRule.
func createDestinationRuleFromHelidonWorkload(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, helidonWorkload *unstructured.Unstructured) (*istioclient.DestinationRule, error) {
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
		return mutateDestinationRuleFromHelidonWorkload(destinationRule, rule, namespace, helidonWorkload)

	}
	return nil, nil
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRuleFromHelidonWorkload(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, namespace *corev1.Namespace, helidonWorkload *unstructured.Unstructured) (*istioclient.DestinationRule, error) {
	dest, err := createDestinationFromRuleOrHelidonWorkload(rule, helidonWorkload)
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

// createDestinationFromRuleOrService creates a destination from the rule or workload
// If the rule contains destination information that is used otherwise workload information is used
func createDestinationFromRuleOrHelidonWorkload(rule vzapi.IngressRule, helidonWorkload *unstructured.Unstructured) (*istio.HTTPRouteDestination, error) {
	if len(rule.Destination.Host) > 0 {

		dest, err := destination.CreateDestinationFromRule(rule)
		if err != nil {
			return nil, err
		}
		return dest, err

	}

	return createDestinationFromHelidonWorkload(helidonWorkload)

}

// creates a destination in the virtual service from helidon workload if port details not present in trait
func createDestinationFromHelidonWorkload(helidonWorkload *unstructured.Unstructured) (*istio.HTTPRouteDestination, error) {

	helidonWorkloadStruct := helidonWorkload.UnstructuredContent()
	spec, found, err := unstructured.NestedMap(helidonWorkloadStruct, "spec")
	if !found || err != nil {
		return nil, errors.New("spec key in a component doesn't exist or not found in the specified type")
	}
	deploymentTemplate, found, err := unstructured.NestedMap(spec, "deploymentTemplate")
	if !found || err != nil {
		return nil, errors.New("deployment template key in a component doesn't exist or not found in the specified type")
	}
	podSpec, found, err := unstructured.NestedMap(deploymentTemplate, "podSpec")
	if !found || err != nil {
		return nil, errors.New("pod spec in a component doesn't exist or not found in the specified type")
	}
	container, found, err := unstructured.NestedSlice(podSpec, "containers")
	if !found || err != nil {
		return nil, errors.New("container in a component doesn't exist or not found in the specified type")
	}
	metaData, found, err := unstructured.NestedMap(deploymentTemplate, "metadata")
	if !found || err != nil {
		return nil, errors.New("metadata in a component doesn't exist or not found in the specified type")
	}
	name, found, err := unstructured.NestedString(metaData, "name")
	if !found || err != nil {
		return nil, errors.New("name in a component doesn't exist or not found in the specified type")
	}

	// Iterate over the container slice
	for _, item := range container {

		// Type assertion to convert the item back to map[string]interface{}
		if itemMap, ok := item.(map[string]interface{}); ok {

			// Check if the "ports" key exists in the map
			if ports, exists := itemMap["ports"]; exists {

				// Type assertion to convert "ports" to []interface{}
				if portsSlice, ok := ports.([]interface{}); ok {

					// Iterate over the ports slice
					for _, port := range portsSlice {

						// Type assertion to convert each port to map[string]interface{}
						if portMap, ok := port.(map[string]interface{}); ok {

							// Check if the "containerPort" key exists in the portMap
							if containerPort, exists := portMap["containerPort"]; exists {

								// Access the value of "containerPort" and convert into uint32
								int32ContainerPort := uint32(containerPort.(int64))

								dest := istio.HTTPRouteDestination{

									Destination: &istio.Destination{Host: name}}

								// Set the port to rule destination port
								dest.Destination.Port = &istio.PortSelector{Number: int32ContainerPort}
								return &dest, nil

							}
						}
					}
				}
			}
		}

	}
	return nil, errors.New("unable to select data for specified destination port")
}
