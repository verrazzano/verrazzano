// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	 networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const systemNamespace = "verrazzano-system"

// GetInstanceInfo returns the instance info for the local install.
func GetInstanceInfo(client client.Client, cr *v1alpha1.Verrazzano) *v1alpha1.InstanceInfo {

	ingressList := &networkingv1.IngressList{}
	err := client.List(context.TODO(), ingressList)
	if err != nil {
		zap.S().Errorf("Error listing ingresses: %v", err)
		return nil
	}
	if len(ingressList.Items) == 0 {
		zap.S().Warn("No ingresses found, unable to build instance info")
		return nil
	}

	// Console ingress always exist. Only show console URL if the console was enabled during install.
	var consoleURL *string
	if cr.Spec.Components.Console == nil || *cr.Spec.Components.Console.Enabled {
		consoleURL = getSystemIngressURL(ingressList.Items, systemNamespace, constants.VzConsoleIngress)
	} else {
		consoleURL = nil
	}
	instanceInfo := &v1alpha1.InstanceInfo{
		ConsoleURL:    consoleURL,
		RancherURL:    getSystemIngressURL(ingressList.Items, "cattle-system", "rancher"),
		KeyCloakURL:   getSystemIngressURL(ingressList.Items, "keycloak", "keycloak"),
		ElasticURL:    getSystemIngressURL(ingressList.Items, systemNamespace, "vmi-system-es-ingest"),
		KibanaURL:     getSystemIngressURL(ingressList.Items, systemNamespace, "vmi-system-kibana"),
		GrafanaURL:    getSystemIngressURL(ingressList.Items, systemNamespace, "vmi-system-grafana"),
		PrometheusURL: getSystemIngressURL(ingressList.Items, systemNamespace, "vmi-system-prometheus"),
	}
	return instanceInfo
}

func getSystemIngressURL(ingresses []networkingv1.Ingress, namespace string, name string) *string {
	var ingress = findIngress(ingresses, namespace, name)
	if ingress == nil {
		zap.S().Infof("No ingress found for %s/%s", namespace, name)
		return nil
	}
	url := fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
	return &url
}

func findIngress(ingresses []networkingv1.Ingress, namespace string, name string) *networkingv1.Ingress {
	for _, ingress := range ingresses {
		if ingress.Name == name && ingress.Namespace == namespace {
			return &ingress
		}
	}
	return nil
}
