// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"net"
)

const (
	nginxExternalIPKey = "controller.service.externalIPs"
	istioExternalIPKey = "gateways.istio-ingressgateway.externalIPs"
	nginxComponentName = "ingress-controller"
	istioComponentName = "istio"
)

//ValidateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
// Validation for nginx and istio installargs when type is set as NodePort
func ValidateForExternalIPSWithNodePort(spec *vzapi.VerrazzanoSpec, compName string) error {
	// If ingress is not set, then type NodePort cannot be set
	// If type is not NodePort further check is not needed
	if spec.Components.Ingress != nil {
		if spec.Components.Ingress.Type != vzapi.NodePort {
			return nil
		}
	}

	switch compName {
	case nginxComponentName:
		if spec.Components.Ingress != nil {
			if spec.Components.Ingress.Type == vzapi.NodePort {
				if spec.Components.Ingress.NGINXInstallArgs != nil {
					return checkArgs(spec.Components.Ingress.NGINXInstallArgs, nginxExternalIPKey, compName)
				}
				// if ingress args are not set at all, then external ips will not be set
				return fmt.Errorf("'nginxInstallArgs' cannot be empty. ExternalIPs needs to be specified here as type is NodePort")
			}
		}
	case istioComponentName:
		if spec.Components.Istio != nil {
			if spec.Components.Istio.IstioInstallArgs != nil {
				return checkArgs(spec.Components.Istio.IstioInstallArgs, istioExternalIPKey, compName)
			}
			if spec.Components.Istio.IstioInstallArgs == nil && spec.Components.Ingress.Type == vzapi.NodePort {
				return fmt.Errorf("'istioInstallArgs' cannot be empty. ExternalIPs needs to be specified here as type is NodePort")
			}
		} else {
			// if istio args are not set at all, also check whether nodePort is set.
			// This is because NodePort is only specified under Ingress Nginx args , which is re-used as a type in Istio
			if spec.Components.Ingress.Type == vzapi.NodePort {
				return fmt.Errorf("Istio component cannot be empty. ExternalIPs needs to be specified here as type is NodePort")
			}
		}
	}
	return nil
}

// This method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func checkArgs(installArgs []vzapi.InstallArgs, argsKeyName, compName string) error {
	var keyPresent bool
	for _, installArg := range installArgs {
		if installArg.Name == argsKeyName {
			keyPresent = true
			if len(installArg.ValueList) < 1 {
				return fmt.Errorf("At least one %s external ips need to be set as an array for the key \"%v\"", compName, installArg.Name)
			}
			if net.ParseIP(installArg.ValueList[0]) == nil {
				return fmt.Errorf("Controller external service key \"%v\" with IP \"%v\" is of invalid format for %s. Must be a proper IP address format", installArg.Name, installArg.ValueList[0], compName)
			}
		}
	}
	if !keyPresent {
		return fmt.Errorf("Key \"%v\" not found for component \"%v\" for type NodePort", argsKeyName, compName)
	}
	return nil
}
