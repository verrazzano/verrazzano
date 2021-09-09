// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetComponents tests getting the components
// GIVEN a component
//  WHEN I call GetComponents
//  THEN the Get returns the correct components
func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetComponents()
	var compsNames []string
	for _, comp := range comps {
		compsNames = append(compsNames, comp.Name())
	}
	assert.Len(comps, 17, "Wrong number of components")
	assert.Contains(compsNames, "istio-base")
	assert.Contains(compsNames, "istiod")
	assert.Contains(compsNames, "istio-ingress")
	assert.Contains(compsNames, "istio-egress")
	assert.Contains(compsNames, "istiocoredns")
	assert.Contains(compsNames, "istio")
	assert.Contains(compsNames, "ingress-controller")
	assert.Contains(compsNames, "cert-manager")
	assert.Contains(compsNames, "external-dns")
	assert.Contains(compsNames, "rancher")
	assert.Contains(compsNames, "verrazzano")
	assert.Contains(compsNames, "coherence-operator")
	assert.Contains(compsNames, "weblogic-operator")
	assert.Contains(compsNames, "oam-kubernetes-runtime")
	assert.Contains(compsNames, "verrazzano-application-operator")
	assert.Contains(compsNames, "mysql")
	assert.Contains(compsNames, "keycloak")
}
