// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetHostnameFromGateway(namespace string, appConfigName string) (string, error) {
	cs, err := GetIstioClientset()
	if err != nil {
		return "", err
	}
	gateways, err := cs.NetworkingV1alpha3().Gateways(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	// if an optional appConfigName is provided, construct the gateway name from the namespace and
	// appConfigName and look for that specific gateway, otherwise just use the first gateway
	gatewayName := ""
	if len(appConfigName) > 0 {
		gatewayName = fmt.Sprintf("%s-%s-gw", namespace, appConfigName)
	}

	for _, gateway := range gateways.Items {
		if len(gatewayName) > 0 && gatewayName != gateway.ObjectMeta.Name {
			continue
		}
		if len(gateway.Spec.Servers) > 0 && len(gateway.Spec.Servers[0].Hosts) > 0 {
			return gateway.Spec.Servers[0].Hosts[0], nil
		}
	}

	// this can happen if the app gateway has not been created yet, the caller should
	// keep retrying and eventually we should get a gateway with a host
	return "", nil
}
