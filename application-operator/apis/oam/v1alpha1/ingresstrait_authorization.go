// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

// AuthorizationRuleFrom includes a list of request principals.
type AuthorizationRuleFrom struct {
	// Specifies the request principals for access to a request.
	RequestPrincipals []string `json:"requestPrincipals,omitempty"`
}

// AuthorizationRuleCondition provides additional required attributes for authorization.
type AuthorizationRuleCondition struct {
	// The name of a request attribute.
	Key string `json:"key,omitempty"`
	// A list of allowed values for the attribute.
	Values []string `json:"values,omitempty"`
}

// AuthorizationRule matches requests from a list of request principals that access a specific path subject to a
// list of conditions.
type AuthorizationRule struct {
	// Specifies the request principals for access to a request. An asterisk (*) will match when the value is not empty,
	// for example, if any request principal is found in the request.
	From *AuthorizationRuleFrom `json:"from,omitempty"`
	// Specifies a list of additional conditions for access to a request.
	// +optional
	When []*AuthorizationRuleCondition `json:"when,omitempty"`
}

// AuthorizationPolicy defines the set of rules for authorizing a request.
type AuthorizationPolicy struct {
	// Rules are used to match requests from request principals to specific paths given an optional list of conditions.
	Rules []*AuthorizationRule `json:"rules,omitempty"`
}
