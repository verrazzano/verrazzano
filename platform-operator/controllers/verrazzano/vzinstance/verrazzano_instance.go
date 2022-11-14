// Copyright (C) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
)

// GetInstanceInfo returns the instance info for the local install.
func GetInstanceInfo(ctx spi.ComponentContext) *v1alpha1.InstanceInfo {
	ingressList := &networkingv1.IngressList{}
	err := ctx.Client().List(context.TODO(), ingressList)
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
	if vzconfig.IsConsoleEnabled(ctx.EffectiveCR()) {
		consoleURL = getComponentIngressURL(ingressList.Items, ctx, authproxy.ComponentName, constants.VzConsoleIngress)
	} else {
		consoleURL = nil
	}

	instanceInfo := &v1alpha1.InstanceInfo{
		ConsoleURL:    consoleURL,
		RancherURL:    getComponentIngressURL(ingressList.Items, ctx, rancher.ComponentName, constants.RancherIngress),
		KeyCloakURL:   getComponentIngressURL(ingressList.Items, ctx, keycloak.ComponentName, constants.KeycloakIngress),
		ElasticURL:    getComponentIngressURL(ingressList.Items, ctx, opensearch.ComponentName, constants.OpensearchIngress),
		KibanaURL:     getComponentIngressURL(ingressList.Items, ctx, opensearchdashboards.ComponentName, constants.KibanaIngress),
		GrafanaURL:    getComponentIngressURL(ingressList.Items, ctx, grafana.ComponentName, constants.GrafanaIngress),
		PrometheusURL: getComponentIngressURL(ingressList.Items, ctx, promoperator.ComponentName, constants.PrometheusIngress),
		KialiURL:      getComponentIngressURL(ingressList.Items, ctx, kiali.ComponentName, constants.KialiIngress),
		JaegerURL:     getComponentIngressURL(ingressList.Items, ctx, jaegeroperator.ComponentName, constants.JaegerIngress),
	}
	return instanceInfo
}

func getComponentIngressURL(ingresses []networkingv1.Ingress, compContext spi.ComponentContext, componentName string, ingressName string) *string {
	found, comp := registry.FindComponent(componentName)
	if !found {
		zap.S().Debugf("No component %s found", componentName)
		return nil
	}
	for _, compIngressName := range comp.GetIngressNames(compContext) {
		if compIngressName.Name == ingressName {
			return getSystemIngressURL(ingresses, compContext, compIngressName.Namespace, compIngressName.Name)
		}
	}
	zap.S().Debugf("No ingress %s found for component %s", ingressName, componentName)
	return nil
}

func getSystemIngressURL(ingresses []networkingv1.Ingress, compContext spi.ComponentContext, namespace string, name string) *string {
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
