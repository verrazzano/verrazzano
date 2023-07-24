// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import securityv1beta1 "istio.io/api/security/v1beta1"

func ConstructAuthPolicyRule(namespaces []string, fromPrincipals []string, toPorts []string) *securityv1beta1.Rule {
	return &securityv1beta1.Rule{
		From: []*securityv1beta1.Rule_From{{
			Source: &securityv1beta1.Source{
				Principals: fromPrincipals,
				Namespaces: namespaces,
			},
		}},
		To: []*securityv1beta1.Rule_To{{
			Operation: &securityv1beta1.Operation{
				Ports: toPorts,
			},
		}},
	}
}
