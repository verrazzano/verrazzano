// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const systemNamespace = "verrazzano-system"

// GetInstanceInfo returns the instance info for the local install.
func GetInstanceInfo(client client.Client) *v1alpha1.InstanceInfo {

	ingressList := &extv1beta1.IngressList{}
	err := client.List(context.TODO(), ingressList)
	if err != nil {
		zap.S().Errorf("Error listing ingresses: %v", err)
		return nil
	}
	if len(ingressList.Items) == 0 {
		zap.S().Errorf("No ingresses found, unable to build instance info")
		return nil
	}

	instanceInfo := &v1alpha1.InstanceInfo{
		Console:       getSystemIngressURL(client, ingressList.Items, systemNamespace, "verrazzano-console-ingress"),
		RancherURL:    getSystemIngressURL(client, ingressList.Items, "cattle-system", "rancher"),
		KeyCloakURL:   getSystemIngressURL(client, ingressList.Items, "keycloak", "keycloak"),
		ElasticURL:    getSystemIngressURL(client, ingressList.Items, systemNamespace, "vmi-system-es-ingest"),
		KibanaURL:     getSystemIngressURL(client, ingressList.Items, systemNamespace, "vmi-system-kibana"),
		GrafanaURL:    getSystemIngressURL(client, ingressList.Items, systemNamespace, "vmi-system-grafana"),
		PrometheusURL: getSystemIngressURL(client, ingressList.Items, systemNamespace, "vmi-system-prometheus"),
	}
	return instanceInfo
}

func getSystemIngressURL(client client.Client, ingresses []extv1beta1.Ingress, namespace string, name string) *string {
	var ingress *extv1beta1.Ingress = findIngress(ingresses, namespace, name)
	if ingress == nil {
		zap.S().Infof("No ingress found for %s/%s", namespace, name)
		return nil
	}
	url := fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
	return &url
}

func findIngress(ingresses []extv1beta1.Ingress, namespace string, name string) *extv1beta1.Ingress {
	for _, ingress := range ingresses {
		if ingress.Name == name && ingress.Namespace == namespace {
			return &ingress
		}
	}
	return nil
}
