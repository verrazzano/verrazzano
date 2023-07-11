// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherenceresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	azp "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/authorizationpolicy"
	gw "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/gateway"
	istio "istio.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
func NewTraitDefaultsForCOHWorkload(workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, error) {
	path := "/metrics"
	port := 9612
	var secret *string

	enabled, p, s, err := fetchCoherenceMetricsSpec(workload)
	if err != nil {
		return nil, err
	}
	if enabled == nil || *enabled {
		if p != nil {
			port = *p
		}
		if s != nil {
			secret = s
		}
	}
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:   &path,
		Secret: secret,
		//Scraper: &r.Scraper
	}, nil
}
func fetchCoherenceMetricsSpec(workload *unstructured.Unstructured) (*bool, *int, *string, error) {
	// determine if metrics is enabled
	enabled, found, err := unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var e *bool
	if found {
		e = &enabled
	}

	// get the metrics port
	port, found, err := unstructured.NestedInt64(workload.Object, "spec", "coherence", "metrics", "port")
	if err != nil {
		return nil, nil, nil, err
	}
	var p *int
	if found {
		p2 := int(port)
		p = &p2
	}

	// get the secret if ssl is enabled
	enabled, found, err = unstructured.NestedBool(workload.Object, "spec", "coherence", "metrics", "ssl", "enabled")
	if err != nil {
		return nil, nil, nil, err
	}
	var s *string
	if found && enabled {
		secret, found, err := unstructured.NestedString(workload.Object, "spec", "coherence", "metrics", "ssl", "secrets")
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			s = &secret
		}
	}
	return e, p, s, nil
}
