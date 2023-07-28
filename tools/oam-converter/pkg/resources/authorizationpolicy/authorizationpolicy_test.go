// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authorizationpolicy

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"istio.io/api/security/v1beta1"
	v1beta12 "istio.io/api/type/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func Test_getIngressTraitNsn(t *testing.T) {
	actual := getIngressTraitNsn("hello", "helidon")
	expected := "hello-helidon"
	assert.Equal(t, expected, actual, "Unexpected result for Namespace: %s, Name: %s", "hello", "helidon")
}

func Test_CreateAuthorizationPolicyRule(t *testing.T) {
	// Test case 1: Valid authorization rule without 'From' clause
	rule1 := &vzapi.AuthorizationRule{
		When: []*vzapi.AuthorizationRuleCondition{
			{Key: "app", Values: []string{"myapp"}},
		},
	}
	path1 := "/api/v1"
	hosts1 := []string{"example.com"}
	requireFrom1 := false

	expectedRule1 := &v1beta1.Rule{
		When: []*v1beta1.Condition{
			{Key: "app", Values: []string{"myapp"}},
		},
		To: []*v1beta1.Rule_To{{
			Operation: &v1beta1.Operation{
				Hosts: hosts1,
				Paths: []string{path1},
			},
		}},
	}

	ruleResult1, err1 := createAuthorizationPolicyRule(rule1, path1, hosts1, requireFrom1)
	assert.Nil(t, err1, "Error was not expected for Test case 1")
	assert.Equal(t, expectedRule1, ruleResult1, "Test case 1 failed: Unexpected result")

	// Test case 2: Authorization rule with missing 'From' clause
	rule2 := &vzapi.AuthorizationRule{
		From: nil,
	}
	path2 := ""
	hosts2 := []string{}
	requireFrom2 := true

	ruleResult2, err2 := createAuthorizationPolicyRule(rule2, path2, hosts2, requireFrom2)
	assert.NotNil(t, err2, "Error was expected for Test case 2")
	assert.Contains(t, err2.Error(), "Authorization Policy requires 'From' clause", "Test case 2 failed: Unexpected error message")
	assert.Nil(t, ruleResult2, "Test case 2 failed: Result should be nil due to error")
}

func TestMutateAuthorizationPolicy(t *testing.T) {
	// Input data
	vzPolicy := &vzapi.AuthorizationPolicy{
		Rules: []*vzapi.AuthorizationRule{
			{
				From: nil,
				When: []*vzapi.AuthorizationRuleCondition{
					{Key: "app", Values: []string{"myapp"}},
				},
			},
			// Add more rules here if needed for testing other cases
		},
	}
	path := "/api/v1"
	hosts := []string{"example.com"}
	requireFrom := false

	// Call the function being tested
	authzPolicy := &clisecurity.AuthorizationPolicy{}
	resultPolicy, err := mutateAuthorizationPolicy(authzPolicy, vzPolicy, path, hosts, requireFrom)

	// Assertions
	assert.Nil(t, err, "Error was not expected")
	assert.NotNil(t, resultPolicy, "Result policy should not be nil")
	assert.Equal(t, 1, len(resultPolicy.Spec.Rules), "Unexpected number of rules in the result policy")

	// Check the contents of the rule
	expectedRule := &v1beta1.Rule{
		When: []*v1beta1.Condition{
			{Key: "app", Values: []string{"myapp"}},
		},
		To: []*v1beta1.Rule_To{{
			Operation: &v1beta1.Operation{
				Hosts: hosts,
				Paths: []string{path},
			},
		}},
	}
	assert.Equal(t, expectedRule, resultPolicy.Spec.Rules[0], "Result rule does not match the expected rule")

	// Check the authzPolicy.Spec.Selector
	expectedSelector := &v1beta12.WorkloadSelector{
		MatchLabels: map[string]string{"istio": "ingressgateway"},
	}
	assert.Equal(t, expectedSelector, resultPolicy.Spec.Selector, "Result selector does not match the expected selector")
}

func TestCreateAuthorizationPolicies(t *testing.T) {
	// Input data
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-ingress-trait",
		},
	}
	rule := vzapi.IngressRule{
		Paths: []vzapi.IngressPath{
			{
				Path: "/api/v1",
				Policy: &vzapi.AuthorizationPolicy{
					Rules: []*vzapi.AuthorizationRule{
						{
							From: &vzapi.AuthorizationRuleFrom{
								RequestPrincipals: []string{"user:john"},
							},
							When: []*vzapi.AuthorizationRuleCondition{
								{Key: "app", Values: []string{"myapp"}},
							},
						},
					},
				},
			},
		},
	}
	namePrefix := "test-policy"
	hosts := []string{"example.com"}

	// Call the function being tested for each path
	var authzPolicies []*clisecurity.AuthorizationPolicy
	for _, path := range rule.Paths {
		authzPolicy, err := CreateAuthorizationPolicies(trait, vzapi.IngressRule{Paths: []vzapi.IngressPath{path}}, namePrefix, hosts)
		assert.NoError(t, err, "Error was not expected")
		authzPolicies = append(authzPolicies, authzPolicy)
	}

	// Assertions
	assert.NotNil(t, authzPolicies, "Result policies should not be nil")

	// Check the number of returned AuthorizationPolicies
	expectedNumPolicies := len(rule.Paths)
	assert.Equal(t, expectedNumPolicies, len(authzPolicies), "Unexpected number of policies returned")

	// Check the contents of each returned AuthorizationPolicy
	for i, authzPolicy := range authzPolicies {
		expectedPolicyName := fmt.Sprintf("test-policy-%s", strings.Replace(rule.Paths[i].Path, "/", "", -1))
		expectedPolicyNamespace := constants.IstioSystemNamespace
		expectedPolicyLabels := map[string]string{
			constants.LabelIngressTraitNsn: getIngressTraitNsn(trait.Namespace, trait.Name),
		}

		assert.Equal(t, "AuthorizationPolicy", authzPolicy.Kind, "Kind does not match")
		assert.Equal(t, consts.AuthorizationAPIVersion, authzPolicy.APIVersion, "APIVersion does not match")
		assert.Equal(t, expectedPolicyName, authzPolicy.ObjectMeta.Name, "Policy name does not match")
		assert.Equal(t, expectedPolicyNamespace, authzPolicy.ObjectMeta.Namespace, "Policy namespace does not match")
		assert.Equal(t, expectedPolicyLabels, authzPolicy.ObjectMeta.Labels, "Policy labels do not match")

		// Check the Rule in the returned AuthorizationPolicy
		assert.Equal(t, 1, len(authzPolicy.Spec.Rules), "Unexpected number of rules in the policy")

	}
}
