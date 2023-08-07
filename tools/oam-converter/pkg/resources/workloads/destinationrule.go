// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloads

import (
	"errors"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationrule"
	serviceDestination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/services"
	"google.golang.org/protobuf/types/known/durationpb"
	istionet "istio.io/api/networking/v1alpha3"
	istio "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"time"
)

// creates or updates the DestinationRule.
func createDestinationRuleFromWorkload(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, helidonWorkload *unstructured.Unstructured, service *corev1.Service) (*istioclient.DestinationRule, error) {
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
		return mutateDestinationRuleFromWorkload(destinationRule, rule, namespace, helidonWorkload, service)

	}
	return nil, nil
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRuleFromWorkload(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, namespace *corev1.Namespace, helidonWorkload *unstructured.Unstructured, service *corev1.Service) (*istioclient.DestinationRule, error) {
	dest, err := createDestinationFromRuleOrService(rule, helidonWorkload, service)
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
func createDestinationFromRuleOrService(rule vzapi.IngressRule, helidonWorkload *unstructured.Unstructured, service *corev1.Service) (*istio.HTTPRouteDestination, error) {
	if len(rule.Destination.Host) > 0 {

		dest, err := destination.CreateDestinationFromRule(rule)
		if err != nil {
			return nil, err
		}
		return dest, err

	}
	if rule.Destination.Port != 0 {
		return serviceDestination.CreateDestinationMatchRulePort(service, rule.Destination.Port)
	}
	if helidonWorkload == nil && service != nil {
		return serviceDestination.CreateDestinationFromService(service)
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
								var int32ContainerPort uint32
								if isInt64(containerPort) {
									int32ContainerPort = uint32(containerPort.(int64))
								}
								if isFloat64(containerPort) {
									int32ContainerPort = uint32(containerPort.(float64))
								}
								// Access the value of "containerPort" and convert into uint32

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
			return nil, errors.New("Port does not exist")
		}

	}
	return nil, errors.New("unable to select data for specified destination port")
}

func isInt64(val interface{}) bool {
	return reflect.TypeOf(val).Kind() == reflect.Int64
}

func isFloat64(val interface{}) bool {
	return reflect.TypeOf(val).Kind() == reflect.Float64
}
