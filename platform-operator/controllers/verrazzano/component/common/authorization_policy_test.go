// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConstructAuthPolicyRule(t *testing.T) {
	asserts := assert.New(t)
	authPolicyRule := ConstructAuthPolicyRule([]string{"ns1"}, []string{"principal1"}, []string{"10080"})
	asserts.Equal([]string{"ns1"}, authPolicyRule.GetFrom()[0].GetSource().Namespaces)
	asserts.Equal([]string{"principal1"}, authPolicyRule.GetFrom()[0].GetSource().Principals)
	asserts.Equal([]string{"10080"}, authPolicyRule.GetTo()[0].GetOperation().Ports)
}
