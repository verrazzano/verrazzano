// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"fmt"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"

	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

// CreateIngressChildResourcesFromHelidon create child resources from helidon workload
func CreateIngressChildResourcesFromHelidon(conversionComponent *types.ConversionComponents, gateway *vsapi.Gateway, allHostsForTrait []string) ([]*vsapi.VirtualService, []*istioclient.DestinationRule, []*clisecurity.AuthorizationPolicy, error) {
	var virtualServices []*vsapi.VirtualService
	var destinationRules []*istioclient.DestinationRule
	var authzPolicies []*clisecurity.AuthorizationPolicy
	if conversionComponent.IngressTrait != nil {
		rules := conversionComponent.IngressTrait.Spec.Rules
		for index, rule := range rules {

			vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, conversionComponent.IngressTrait, conversionComponent.AppName, conversionComponent.AppNamespace)

			if err != nil {
				print(err)
				return nil, nil, nil, err
			}

			vsName := fmt.Sprintf("%s-rule-%d-vs", conversionComponent.IngressTrait.Name, index)
			drName := fmt.Sprintf("%s-rule-%d-dr", conversionComponent.ComponentName, index)
			authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", conversionComponent.ComponentName, index)
			virtualService, err := createVirtualServiceFromHelidonWorkload(conversionComponent.AppNamespace, rule, vsHosts, vsName, gateway, conversionComponent.Helidonworkload)
			if err != nil {
				return nil, nil, nil, err
			}
			virtualServices = append(virtualServices, virtualService)
			destinationRule, err := createDestinationRuleFromHelidonWorkload(conversionComponent.IngressTrait, rule, drName, conversionComponent.Helidonworkload)
			if err != nil {
				return nil, nil, nil, err
			}
			destinationRules = append(destinationRules, destinationRule)
			authzPolicy, err := azp.CreateAuthorizationPolicies(conversionComponent.IngressTrait, rule, authzPolicyName, allHostsForTrait)
			if err != nil {
				return nil, nil, nil, err
			}
			authzPolicies = append(authzPolicies, authzPolicy)

		}
		return virtualServices, destinationRules, authzPolicies, nil
	}
	return nil, nil, nil, nil
}
