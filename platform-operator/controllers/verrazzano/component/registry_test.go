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

	assert.Len(comps, 18, "Wrong number of components")
	assert.Equal(comps[0].Name(), "istio-base")
	assert.Equal(comps[1].Name(), "istiod")
	assert.Equal(comps[2].Name(), "istio-ingress")
	assert.Equal(comps[3].Name(), "istio-egress")
	assert.Equal(comps[4].Name(), "istiocoredns")
	assert.Equal(comps[5].Name(), "grafana")
	assert.Equal(comps[6].Name(), "prometheus")
	assert.Equal(comps[7].Name(), "ingress-controller")
	assert.Equal(comps[8].Name(), "cert-manager")
	assert.Equal(comps[9].Name(), "external-dns")
	assert.Equal(comps[10].Name(), "rancher")
	assert.Equal(comps[11].Name(), "verrazzano")
	assert.Equal(comps[12].Name(), "coherence-operator")
	assert.Equal(comps[13].Name(), "weblogic-operator")
	assert.Equal(comps[14].Name(), "oam-kubernetes-runtime")
	assert.Equal(comps[15].Name(), "verrazzano-application-operator")
	assert.Equal(comps[16].Name(), "mysql")
	assert.Equal(comps[17].Name(), "keycloak")
}
