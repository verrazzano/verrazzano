// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"fmt"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationRule"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	vs "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/virtualservice"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/workloads"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
)

func CreateResources(conversionComponents []*types.ConversionComponents) (*types.KubeResources, error) {
	var virtualServices []*vsapi.VirtualService
	var destinationRules []*istioclient.DestinationRule
	var authzPolicies []*clisecurity.AuthorizationPolicy
	var virtualService []*vsapi.VirtualService
	var destinationRule []*istioclient.DestinationRule
	var authzPolicy []*clisecurity.AuthorizationPolicy
	outputResources := types.KubeResources{}

	gateway, allHostsForTrait, err := gw.CreateGatewayResource(conversionComponents)
	if err != nil {
		return nil, err
	}
	listGateway, err := gw.CreateListGateway(gateway)
	if err != nil {
		return nil, err
	}

	for _, conversionComponent := range conversionComponents {
		if conversionComponent.Weblogicworkload != nil || conversionComponent.Coherenceworkload != nil {

			virtualService, destinationRule, authzPolicy, err = createChildResources(conversionComponent, gateway, allHostsForTrait)
			if err != nil {
				return nil, fmt.Errorf("failed to create Child resources from Weblogic workload %w", err)

			}
			virtualServices = append(virtualServices, virtualService...)
			destinationRules = append(destinationRules, destinationRule...)
			authzPolicies = append(authzPolicies, authzPolicy...)

		}
		if conversionComponent.Helidonworkload != nil || conversionComponent.Service != nil {
			virtualService, destinationRule, authzPolicy, err = workloads.CreateIngressChildResourcesFromWorkload(conversionComponent, gateway, allHostsForTrait)
			if err != nil {
				return nil, fmt.Errorf("failed to create Child resources from workload %w", err)

			}
			virtualServices = append(virtualServices, virtualService...)
			destinationRules = append(destinationRules, destinationRule...)
			authzPolicies = append(authzPolicies, authzPolicy...)

		}
	}
	//Appending it to Kube Resources to print the output
	outputResources.DestinationRules = destinationRules
	outputResources.AuthPolicies = authzPolicies
	outputResources.VirtualServices = virtualServices
	outputResources.Gateway = listGateway
	return &outputResources, nil
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
	return nil, nil, nil, fmt.Errorf("ingress Trait is empty")
}
