// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherenceresources

import (
	"fmt"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

func CreateIngressChildResourcesFromCoherence(conversionComponent *types.ConversionComponents) error {
	rules := conversionComponent.IngressTrait.Spec.Rules

	// If there are no rules, create a single default rule
	gateway, allHostsForTrait, err := gw.CreateCertificateAndSecret(conversionComponent)
	if err != nil {
		return err
	}
	for index, rule := range rules {

		// Find the services associated with the trait in the application configuration.
		vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, conversionComponent.IngressTrait, conversionComponent.AppName, conversionComponent.AppNamespace)
		if err != nil {
			print(err)
			return err
		}
		vsName := fmt.Sprintf("%s-rule-%d-vs", conversionComponent.IngressTrait.Name, index)
		drName := fmt.Sprintf("%s-rule-%d-dr", conversionComponent.IngressTrait.Name, index)
		authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", conversionComponent.IngressTrait.Name, index)
		err = createVirtualServiceFromCoherenceWorkload(conversionComponent.IngressTrait, rule, vsHosts, vsName, gateway, conversionComponent.Coherenceworkload)
		if err != nil {
			return err
		}
		err = createDestinationRuleFromCoherenceWorkload(conversionComponent.IngressTrait, rule, drName, conversionComponent.Coherenceworkload)
		if err != nil {
			return err
		}
		err = azp.CreateAuthorizationPolicies(conversionComponent.IngressTrait, rule, authzPolicyName, allHostsForTrait)
		if err != nil {
			return err
		}
	}
	return nil
}
