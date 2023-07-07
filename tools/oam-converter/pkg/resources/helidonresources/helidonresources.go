// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
)

// create child resources from helidon workload
func CreateIngressChildResourcesFromHelidon(traitName string, ingressTrait *vzapi.IngressTrait, helidonWorkload *vzapi.VerrazzanoHelidonWorkload) error {
	rules := ingressTrait.Spec.Rules
	// If there are no rules, create a single default rule
	if len(rules) == 0 {
		rules = []vzapi.IngressRule{{}}
	}
	// Create a list of unique hostnames across all rules in the trait
	allHostsForTrait, err := coallateHosts.CoallateAllHostsForTrait(ingressTrait)
	if err != nil {
		print(err)
		return err
	}
	// Generate the certificate and secret for all hosts in the trait rules
	secretName := gw.CreateGatewaySecret(ingressTrait, allHostsForTrait)
	if secretName != "" {
		gwName, err := gw.BuildGatewayName(ingressTrait)
		if err != nil {
			print(err)
			return err
		}
		gateway, err := gw.CreateGateway(traitName, ingressTrait, allHostsForTrait, gwName, secretName)
		if err != nil {
			print(err)
			return err
		}
		for index, rule := range rules {

			vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, ingressTrait)
			if err != nil {
				print(err)
				return err
			}
			vsName := fmt.Sprintf("%s-rule-%d-vs", ingressTrait.Name, index)
			drName := fmt.Sprintf("%s-rule-%d-dr", ingressTrait.Name, index)
			authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", ingressTrait.Name, index)
			err = createVirtualServiceFromHelidonWorkload(ingressTrait, rule, vsHosts, vsName, gateway, helidonWorkload)
			if err != nil {
				return err
			}
			err = createDestinationRuleFromHelidonWorkload(ingressTrait, rule, drName, helidonWorkload)
			if err != nil {
				return err
			}
			err = azp.CreateAuthorizationPolicies(ingressTrait, rule, authzPolicyName, allHostsForTrait)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
