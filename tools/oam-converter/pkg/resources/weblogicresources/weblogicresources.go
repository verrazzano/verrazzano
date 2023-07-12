// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogicresources

import (
	"fmt"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

// Create child resources from weblogic workload
func CreateIngressChildResourcesFromWeblogic(conversionComponent *types.ConversionComponents) error {
	rules := conversionComponent.IngressTrait.Spec.Rules

	gateway, allHostsForTrait, err := gw.CreateCertificateAndSecret(conversionComponent)
	if err != nil {
		return err
	}
	for index, rule := range rules {
		// Find the services associated with the trait in the application configuration.

		vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, conversionComponent.IngressTrait, conversionComponent.ComponentName, conversionComponent.AppNamespace)
		if err != nil {
			print(err)
			return err
		}
		vsName := fmt.Sprintf("%s-rule-%d-vs", conversionComponent.ComponentName, index)
		drName := fmt.Sprintf("%s-rule-%d-dr", conversionComponent.ComponentName, index)
		authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", conversionComponent.ComponentName, index)
		err = createVirtualServiceFromWeblogicWorkload(conversionComponent.IngressTrait, rule, vsHosts, vsName, gateway, conversionComponent.WeblogicworkloadMap[conversionComponent.ComponentName])
		if err != nil {
			return err
		}
		err = createDestinationRuleFromWeblogicWorkload(conversionComponent.IngressTrait, rule, drName, conversionComponent.WeblogicworkloadMap[conversionComponent.ComponentName])
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
