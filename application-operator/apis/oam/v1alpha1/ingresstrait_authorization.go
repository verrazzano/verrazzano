// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

// AuthorizationRuleFrom includes a list of request principals.
type AuthorizationRuleFrom struct {
	RequestPrincipals []string `json:"requestPrincipals,omitempty"`
}

// AuthorizationRuleCondition specifies additional required attributes.
type AuthorizationRuleCondition struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"values,omitempty"`
}

// AuthorizationRule matches requests from a list of request principals that access a specific path subject to a
// list of conditions.
type AuthorizationRule struct {
	From *AuthorizationRuleFrom        `json:"from,omitempty"`
	When []*AuthorizationRuleCondition `json:"when,omitempty"`
}

// AuthorizationPolicy defines the set of rules for authorizing a request.
type AuthorizationPolicy struct {
	// A list of rules to match the request. A match occurs when at least
	// one rule matches the request.
	Rules []*AuthorizationRule `json:"rules,omitempty"`
}
