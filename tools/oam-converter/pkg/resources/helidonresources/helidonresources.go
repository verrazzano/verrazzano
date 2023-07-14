// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"fmt"

	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

// create child resources from helidon workload
func CreateIngressChildResourcesFromHelidon(conversionComponent *types.ConversionComponents) error {
	rules := conversionComponent.IngressTrait.Spec.Rules
	gateway, allHostsForTrait, err := gw.CreateCertificateAndSecret(conversionComponent)
	if err != nil {
		return err
	}
	for index, rule := range rules {

		vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, conversionComponent.IngressTrait, conversionComponent.AppName, conversionComponent.AppNamespace)

		if err != nil {
			print(err)
			return err
		}

		vsName := fmt.Sprintf("%s-rule-%d-vs", conversionComponent.ComponentName, index)
		drName := fmt.Sprintf("%s-rule-%d-dr", conversionComponent.ComponentName, index)
		authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", conversionComponent.ComponentName, index)
		err = createVirtualServiceFromHelidonWorkload(conversionComponent.IngressTrait, rule, vsHosts, vsName, gateway, conversionComponent.Helidonworkload)
		if err != nil {
			return err
		}
		err = createDestinationRuleFromHelidonWorkload(conversionComponent.IngressTrait, rule, drName, conversionComponent.Helidonworkload)
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
