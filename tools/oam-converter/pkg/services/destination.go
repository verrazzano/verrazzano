// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package services

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	istio "istio.io/api/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// CreateDestinationFromService selects a Service and creates a virtual service destination for the selected service.
// If the selected service does not have a port, it is not included in the destination. If the selected service
// declares port(s), it selects the appropriate one and add it to the destination.
func CreateDestinationFromService(service *corev1.Service) (*istio.HTTPRouteDestination, error) {

	dest := istio.HTTPRouteDestination{
		Destination: &istio.Destination{Host: service.Name}}
	// If the selected service declares port(s), select the appropriate port and add it to the destination.
	if len(service.Spec.Ports) > 0 {
		selectedPort, err := selectPortForDestination(service)
		if err != nil {
			return nil, err
		}
		dest.Destination.Port = &istio.PortSelector{Number: uint32(selectedPort.Port)}
	}
	return &dest, nil
}

// selectPortForDestination selects a Service port to be used for virtual service destination port.
// The port is selected based on the following logic:
//   - If there is one port, return that port.
//   - If there are multiple ports, select the http/WebLogic port.
//   - If there are multiple ports and more than one http/WebLogic port, return an error.
//   - If there are multiple ports and none of then are http/WebLogic ports, return an error.
func selectPortForDestination(service *corev1.Service) (corev1.ServicePort, error) {
	servicePorts := service.Spec.Ports
	// If there is only one port, return that port
	if len(servicePorts) == 1 {
		return servicePorts[0], nil
	}
	allowedPorts := append(getHTTPPorts(service), getWebLogicPorts(service)...)
	// If there are multiple ports and one http/WebLogic port, return that port
	if len(servicePorts) > 1 && len(allowedPorts) == 1 {
		return allowedPorts[0], nil
	}
	// If there are multiple ports and none of them are http/WebLogic ports, return an error
	if len(servicePorts) > 1 && len(allowedPorts) < 1 {
		return corev1.ServicePort{}, fmt.Errorf("unable to select the service port for destination. The service port " +
			"should be named with prefix \"http\" if there are multiple ports OR the IngressTrait must specify the port")
	}
	// If there are multiple http/WebLogic ports, return an error
	if len(allowedPorts) > 1 {
		return corev1.ServicePort{}, fmt.Errorf("unable to select the service port for destination. Only one service " +
			"port should be named with prefix \"http\" OR the IngressTrait must specify the port")
	}
	return corev1.ServicePort{}, fmt.Errorf("unable to select default port for destination")
}

// getHTTPPorts returns all the service ports having the prefix "http" in their names.
func getHTTPPorts(service *corev1.Service) []corev1.ServicePort {
	var httpPorts []corev1.ServicePort
	for _, servicePort := range service.Spec.Ports {
		// Check if service port name has the http prefix
		if strings.HasPrefix(servicePort.Name, "http") {
			httpPorts = append(httpPorts, servicePort)
		}
	}
	return httpPorts
}

// getWebLogicPorts returns WebLogic ports if any present for the service. A port is evaluated as a WebLogic port if
// the port name is from the known WebLogic non-http prefixed port names used by the WebLogic operator.
func getWebLogicPorts(service *corev1.Service) []corev1.ServicePort {
	var webLogicPorts []corev1.ServicePort
	selectorMap := service.Spec.Selector
	value, ok := selectorMap["weblogic.createdByOperator"]
	if !ok || value == "false" {
		return webLogicPorts
	}
	for _, servicePort := range service.Spec.Ports {
		// Check if service port name is one of the predefined WebLogic port names
		for _, webLogicPortName := range constants.WeblogicPortNames {
			if servicePort.Name == webLogicPortName {
				webLogicPorts = append(webLogicPorts, servicePort)
			}
		}
	}
	return webLogicPorts
}

// CreateDestinationMatchRulePort fetches a Service matching the specified rule port and creates virtual service destination.
func CreateDestinationMatchRulePort(service *corev1.Service, rulePort uint32) (*istio.HTTPRouteDestination, error) {

	dest := &istio.HTTPRouteDestination{
		Destination: &istio.Destination{Host: service.Name}}
	// Set the port to rule destination port
	dest.Destination.Port = &istio.PortSelector{Number: rulePort}

	return nil, fmt.Errorf("unable to select service for specified destination port %d", rulePort)
}
