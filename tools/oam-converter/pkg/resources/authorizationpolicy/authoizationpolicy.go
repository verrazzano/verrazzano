// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authorizationpolicy

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"istio.io/api/security/v1beta1"
	v1beta12 "istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// creates Authorization Policy
func CreateAuthorizationPolicies(trait *vzapi.IngressTrait, rule vzapi.IngressRule, namePrefix string, hosts []string) (*clisecurity.AuthorizationPolicy, error) {

	// If any path needs an AuthorizationPolicy then add one for every path
	var addAuthPolicy bool
	for _, path := range rule.Paths {
		if path.Policy != nil {
			addAuthPolicy = true
		}
	}
	for _, path := range rule.Paths {
		if addAuthPolicy {
			requireFrom := true

			// Add a policy rule if one is missing
			if path.Policy == nil {
				path.Policy = &vzapi.AuthorizationPolicy{
					Rules: []*vzapi.AuthorizationRule{{}},
				}
				// No from field required, this is just a path being added
				requireFrom = false
			}

			pathSuffix := strings.Replace(path.Path, "/", "", -1)
			policyName := namePrefix
			if pathSuffix != "" {
				policyName = fmt.Sprintf("%s-%s", policyName, pathSuffix)
			}

			authzPolicy := &clisecurity.AuthorizationPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AuthorizationPolicy",
					APIVersion: consts.AuthorizationAPIVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: constants.IstioSystemNamespace,
					Labels:    map[string]string{constants.LabelIngressTraitNsn: getIngressTraitNsn(trait.Namespace, trait.Name)},
				},
			}
			return mutateAuthorizationPolicy(authzPolicy, path.Policy, path.Path, hosts, requireFrom)
		}
	}
	return nil, nil
}

// mutateAuthorizationPolicy changes the destination rule based upon a trait's configuration
func mutateAuthorizationPolicy(authzPolicy *clisecurity.AuthorizationPolicy, vzPolicy *vzapi.AuthorizationPolicy, path string, hosts []string, requireFrom bool) (*clisecurity.AuthorizationPolicy, error) {
	policyRules := make([]*v1beta1.Rule, len(vzPolicy.Rules))
	var err error
	for i, authzRule := range vzPolicy.Rules {
		policyRules[i], err = createAuthorizationPolicyRule(authzRule, path, hosts, requireFrom)
		if err != nil {
			print(err)
			return nil, err
		}
	}

	authzPolicy.Spec = v1beta1.AuthorizationPolicy{
		Selector: &v1beta12.WorkloadSelector{
			MatchLabels: map[string]string{"istio": "ingressgateway"},
		},
		Rules: policyRules,
	}

	return authzPolicy, nil
}

// createAuthorizationPolicyRule uses the provided information to create an istio authorization policy rule
func createAuthorizationPolicyRule(rule *vzapi.AuthorizationRule, path string, hosts []string, requireFrom bool) (*v1beta1.Rule, error) {
	authzRule := v1beta1.Rule{}

	if requireFrom && rule.From == nil {
		return nil, fmt.Errorf("Authorization Policy requires 'From' clause")
	}
	if rule.From != nil {
		authzRule.From = []*v1beta1.Rule_From{
			{Source: &v1beta1.Source{
				RequestPrincipals: rule.From.RequestPrincipals},
			},
		}
	}

	if len(path) > 0 {
		authzRule.To = []*v1beta1.Rule_To{{
			Operation: &v1beta1.Operation{
				Hosts: hosts,
				Paths: []string{path},
			},
		}}
	}

	if rule.When != nil {
		conditions := []*v1beta1.Condition{}
		for _, vzCondition := range rule.When {
			condition := &v1beta1.Condition{
				Key:    vzCondition.Key,
				Values: vzCondition.Values,
			}
			conditions = append(conditions, condition)
		}
		authzRule.When = conditions
	}

	return &authzRule, nil
}

// Get Ingress Trait Namespace and name appended with "-"
func getIngressTraitNsn(namespace string, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}
