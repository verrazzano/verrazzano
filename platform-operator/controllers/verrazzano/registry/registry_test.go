// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"

	"github.com/stretchr/testify/assert"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profileDir = "../../../manifests/profiles"
)

type fooComponent struct {
	helmcomp.HelmComponent
}

func (f fooComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

var _ spi.Component = &fooComponent{}

// TestGetComponents tests getting the components
// GIVEN a component
//  WHEN I call GetComponents
//  THEN the Get returns the correct components
func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := VzComponentRegistry{}.GetComponents()

	assert.Len(comps, 13, "Wrong number of components")
	assert.Equal(comps[0].Name(), istio.ComponentName)
	assert.Equal(comps[1].Name(), nginx.ComponentName)
	assert.Equal(comps[2].Name(), certmanager.ComponentName)
	assert.Equal(comps[3].Name(), externaldns.ComponentName)
	assert.Equal(comps[4].Name(), rancher.ComponentName)
	assert.Equal(comps[5].Name(), verrazzano.ComponentName)
	assert.Equal(comps[6].Name(), coherence.ComponentName)
	assert.Equal(comps[7].Name(), weblogic.ComponentName)
	assert.Equal(comps[8].Name(), oam.ComponentName)
	assert.Equal(comps[9].Name(), appoper.ComponentName)
	assert.Equal(comps[10].Name(), mysql.ComponentName)
	assert.Equal(comps[11].Name(), keycloak.ComponentName)
	assert.Equal(comps[12].Name(), kiali.ComponentName)
}

// TestFindComponent tests FindComponent
// GIVEN a component
//  WHEN I call FindComponent
//  THEN the true and the component are returned, false and an empty comp otherwise
func TestFindComponent(t *testing.T) {
	found, comp := VzComponentRegistry{}.FindComponent(istio.ComponentName)
	assert.True(t, found)
	assert.NotNil(t, comp)
	assert.Equal(t, istio.ComponentName, comp.Name())
}

// TestRegistryDependencies tests the default Registry components for cycles
// GIVEN a component
//  WHEN I call checkDependencies for it
//  THEN No error is returned that indicates a cycle in the chain
func TestRegistryDependencies(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	registry := VzComponentRegistry{}
	for _, comp := range registry.GetComponents() {
		fakeContext := spi.NewFakeContextWithRegistry(client, &v1alpha1.Verrazzano{}, registry, false, profileDir)
		_, err := spi.CheckDependencies(comp, fakeContext, make(map[string]bool), make(map[string]bool))
		assert.NoError(t, err)
	}
}
