// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
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
		zap.S().Debugf("No ingresses found, unable to build instance info")
		return nil
	}

	// Console ingress always exist. Only show console URL if the console was enabled during install.

	var consoleURL *string
	if cr.Spec.Components.Console == nil || *cr.Spec.Components.Console.Enabled {
		consoleURL = getVerrazzanoIngressURL(ingressList.Items, cr, constants.VzConsoleIngress)
	} else {
		consoleURL = nil
	}

	instanceInfo := &v1alpha1.InstanceInfo{
		ConsoleURL:    consoleURL,
		RancherURL:    getComponentIngressURL(ingressList.Items, cr, rancher.ComponentName),
		KeyCloakURL:   getComponentIngressURL(ingressList.Items, cr, keycloak.ComponentName),
		ElasticURL:    getVerrazzanoIngressURL(ingressList.Items, cr, constants.ElasticsearchIngress),
		KibanaURL:     getVerrazzanoIngressURL(ingressList.Items, cr, constants.KibanaIngress),
		GrafanaURL:    getVerrazzanoIngressURL(ingressList.Items, cr, constants.GrafanaIngress),
		PrometheusURL: getVerrazzanoIngressURL(ingressList.Items, cr, constants.PrometheusIngress),
		KialiURL:      getComponentIngressURL(ingressList.Items, cr, kiali.ComponentName),
	}
	return instanceInfo
}

func getVerrazzanoIngressURL(ingresses []networkingv1.Ingress, cr *v1alpha1.Verrazzano, ingressName string) *string {
	found, comp := registry.FindComponent(verrazzano.ComponentName)
	if !found {
		zap.S().Warnf("No component %s found", verrazzano.ComponentName)
		return nil
	}

	for _, compIngressName := range comp.GetIngressNames(cr) {
		if compIngressName.Name == ingressName {
			return getSystemIngressURL(ingresses, compIngressName.Namespace, compIngressName.Name)
		}
	}
	zap.S().Debugf("No Verrazzano ingress %s found", ingressName)
	return nil
}

func getComponentIngressURL(ingresses []networkingv1.Ingress, cr *v1alpha1.Verrazzano, componentName string) *string {
	found, comp := registry.FindComponent(componentName)
	if !found {
		zap.S().Debugf("No component %s found", componentName)
		return nil
	}
	ingNames := comp.GetIngressNames(cr)
	if len(ingNames) == 0 {
		zap.S().Debugf("No ingress found for component %s", componentName)
		return nil
	}
	return getSystemIngressURL(ingresses, ingNames[0].Namespace, ingNames[0].Name)
}


func getSystemIngressURL(ingresses []networkingv1.Ingress, namespace string, name string) *string {
	var ingress = findIngress(ingresses, namespace, name)
	if ingress == nil {
		zap.S().Debugf("No ingress found for %s/%s", namespace, name)
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
