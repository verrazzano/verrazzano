// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"context"
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetExternalIP Returns the ingress IP of the service given the service type, name, and namespace
func GetExternalIP(client client.Client, serviceType vzapi.IngressType, name, namespace string) (string, error) {
	// Default for NodePort services
	// - On MAC and Windows, container IP is not accessible.  Port forwarding from 127.0.0.1 to container IP is needed.
	externalIP := "127.0.0.1"
	if serviceType == vzapi.LoadBalancer || serviceType == vzapi.NodePort {
		svc := v1.Service{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &svc); err != nil {
			return "", fmt.Errorf("Error getting service %s/%s: %v", namespace, name, err)
		}
		// If externalIPs exists, use it; else use IP from status
		if len(svc.Spec.ExternalIPs) > 0 {
			// In case of OLCNE, the Status.LoadBalancer.Ingress field will be empty, so use the external IP if present
			externalIP = svc.Spec.ExternalIPs[0]
		} else if len(svc.Status.LoadBalancer.Ingress) > 0 {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
		} else {
			return "", fmt.Errorf("No IP found for service %v with type %v", svc.Name, serviceType)
		}
	}
	return externalIP, nil
}
