package coherenceresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	istio "istio.io/api/networking/v1beta1"
)

func CreateIngressChildResourcesFromCoherence(traitName string, ingressTrait *vzapi.IngressTrait, cohereneWorkload *vzapi.VerrazzanoCoherenceWorkload) error {
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
			// Find the services associated with the trait in the application configuration.

			vsHosts, err := coallateHosts.CreateHostsFromIngressTraitRule(rule, ingressTrait)
			if err != nil {
				print(err)
				return err
			}
			vsName := fmt.Sprintf("%s-rule-%d-vs", ingressTrait.Name, index)
			drName := fmt.Sprintf("%s-rule-%d-dr", ingressTrait.Name, index)
			authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", ingressTrait.Name, index)
			err = createVirtualServiceFromCoherenceWorkload(ingressTrait, rule, vsHosts, vsName, gateway, cohereneWorkload)
			if err != nil {
				return err
			}
			err = createDestinationRuleFromCoherenceWorkload(ingressTrait, rule, drName, cohereneWorkload)
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

// createDestinationFromRuleOrService creates a destination from either the rule or the service.
// If the rule contains destination information that is used.
func createDestinationFromRuleOrCoherenceWorkload(rule vzapi.IngressRule, coherenceWorkload *vzapi.VerrazzanoCoherenceWorkload) (*istio.HTTPRouteDestination, error) {
	if len(rule.Destination.Host) > 0 {
		dest := &istio.HTTPRouteDestination{Destination: &istio.Destination{Host: rule.Destination.Host}}
		if rule.Destination.Port != 0 {
			dest.Destination.Port = &istio.PortSelector{Number: rule.Destination.Port}
		}
		return dest, nil
	}

	return nil, nil

}
