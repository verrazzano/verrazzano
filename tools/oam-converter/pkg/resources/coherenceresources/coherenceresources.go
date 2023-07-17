// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherenceresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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
