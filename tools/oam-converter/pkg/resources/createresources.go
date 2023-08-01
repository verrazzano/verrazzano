// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"errors"
	"fmt"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationRule"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/helidonresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/metrics"
	vs "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/virtualservice"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
)
//array that stores all kubernetes resources being outputed by the converter
var kubeResources []any
func CreateResources(conversionComponents []*types.ConversionComponents) ([]any, error) {

	gateway, allHostsForTrait, err := gw.CreateGatewayResource(conversionComponents)
	if err != nil {
		return nil, err
	}
	listGateway, err := gw.CreateListGateway(gateway)
	if err != nil {
		return nil, err
	}
	kubeResources = append(kubeResources, listGateway)
	for _, conversionComponent := range conversionComponents {
		if conversionComponent.MetricsTrait != nil {

			serviceMonitor, err := metrics.CreateMetricsChildResources(conversionComponent)
			kubeResources = append(kubeResources, serviceMonitor)
			if err != nil {
				return nil, errors.New("failed to create service monitor from MetricsTrait")

			}
			continue
		}
		if conversionComponent.Weblogicworkload != nil {

			virtualService, destinationRule, authzPolicy, err := createChildResources(conversionComponent, gateway, allHostsForTrait)
			if err != nil {
				return nil, errors.New("failed to create Child resources from Weblogic workload")

			}
			addResourcesToKubeResources(virtualService, destinationRule, authzPolicy)
		}
		if conversionComponent.Coherenceworkload != nil {
			virtualService, destinationRule, authzPolicy, err := createChildResources(conversionComponent, gateway, allHostsForTrait)
			if err != nil {
				return nil, errors.New("failed to create Child resources from Coherence workload")

			}
			addResourcesToKubeResources(virtualService, destinationRule, authzPolicy)
		}
		if conversionComponent.Helidonworkload != nil {
			virtualService, destinationRule, authzPolicy, err := helidonresources.CreateIngressChildResourcesFromHelidon(conversionComponent, gateway, allHostsForTrait)
			if err != nil {
				return nil, errors.New("failed to create Child resources from Helidon workload")

			}
			addResourcesToKubeResources(virtualService, destinationRule, authzPolicy)
		}
	}

	return kubeResources, nil
}
func addResourcesToKubeResources(virtualService []*vsapi.VirtualService, destinationRule []*istioclient.DestinationRule, authzPolicy []*clisecurity.AuthorizationPolicy){
	for _,index := range virtualService {
		kubeResources = append(kubeResources, index)
	}
	for _,index := range destinationRule {
		kubeResources = append(kubeResources, index)
	}
	for _,index := range authzPolicy {
		kubeResources = append(kubeResources, index)
	}
}
func createChildResources(conversionComponent *types.ConversionComponents, gateway *vsapi.Gateway, allHostsForTrait []string) ([]*vsapi.VirtualService, []*istioclient.DestinationRule, []*clisecurity.AuthorizationPolicy, error) {
	if conversionComponent.IngressTrait != nil {
		rules := conversionComponent.IngressTrait.Spec.Rules
		var virtualServices []*vsapi.VirtualService
		var destinationRules []*istioclient.DestinationRule
		var authzPolicies []*clisecurity.AuthorizationPolicy
		for index, rule := range rules {

			// Find the services associated with the trait in the application configuration.
			vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, conversionComponent.IngressTrait, conversionComponent.AppName, conversionComponent.AppNamespace)

			if err != nil {
				print(err)
				return nil, nil, nil, err
			}
			vsName := fmt.Sprintf("%s-rule-%d-vs", conversionComponent.IngressTrait.Name, index)
			drName := fmt.Sprintf("%s-rule-%d-dr", conversionComponent.IngressTrait.Name, index)
			authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", conversionComponent.IngressTrait.Name, index)
			virtualService, err := vs.CreateVirtualService(conversionComponent.IngressTrait, rule, vsHosts, vsName, gateway)
			virtualServices = append(virtualServices, virtualService)

			if err != nil {
				return nil, nil, nil, err
			}
			destinationRule, err := destination.CreateDestinationRule(conversionComponent.IngressTrait, rule, drName)
			destinationRules = append(destinationRules, destinationRule)
			if err != nil {
				return nil, nil, nil, err
			}
			authzPolicy, err := azp.CreateAuthorizationPolicies(conversionComponent.IngressTrait, rule, authzPolicyName, allHostsForTrait)
			if err != nil {
				return nil, nil, nil, err
			}
			authzPolicies = append(authzPolicies, authzPolicy)
		}
		return virtualServices, destinationRules, authzPolicies, nil
	}
	return nil, nil, nil, errors.New("ingress Trait is empty")
}
