// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestComponentDependenciesNotMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if any depdencies are not met
func TestComponentDependenciesNotMet(t *testing.T) {
	comp := fakeComponent{
		name:         "foo",
		dependencies: []string{"istio"},
		enabled:      true,
	}
	registry := fakeRegistry{
		components: []Component{
			&comp,
			&fakeComponent{
				name:    "istio",
				enabled: true,
				ready:   false,
			},
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContextWithRegistry(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, &registry, false))
	assert.False(t, ready)
}

// TestComponentDependenciesDependencyChartNotInstalled tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if the dependent chart isn't installed
func TestComponentDependenciesDependencyChartNotInstalled(t *testing.T) {
	comp := fakeComponent{
		name:         "foo",
		dependencies: []string{"istio"},
		enabled:      true,
	}
	registry := fakeRegistry{
		components: []Component{
			&comp,
			&fakeComponent{
				name:    "istio",
				enabled: true,
				ready:   false,
			},
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContextWithRegistry(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, &registry, false))
	assert.False(t, ready)
}

// TestComponentMultipleDependenciesPartiallyMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if any depdencies are not met
func TestComponentMultipleDependenciesPartiallyMet(t *testing.T) {
	comp := fakeComponent{
		name:         "foo",
		dependencies: []string{"istio", "cert-manager"},
		enabled:      true,
	}

	registry := fakeRegistry{
		components: []Component{
			&comp,
			&fakeComponent{
				name:    "istio",
				enabled: true,
				ready:   true,
			},
			&fakeComponent{
				name:    "cert-manager",
				enabled: true,
				ready:   false,
			},
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContextWithRegistry(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, &registry, false))
	assert.False(t, ready)
}

// TestComponentMultipleDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the true is returned if all depdencies are met
func TestComponentMultipleDependenciesMet(t *testing.T) {
	comp := fakeComponent{
		name:         "foo",
		dependencies: []string{"istio", "cert-manager"},
		enabled:      true,
	}

	registry := fakeRegistry{
		components: []Component{
			&comp,
			&fakeComponent{
				name:    "istio",
				enabled: true,
				ready:   true,
			},
			&fakeComponent{
				name:    "cert-manager",
				enabled: true,
				ready:   true,
			},
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContextWithRegistry(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, &registry, false))
	assert.True(t, ready)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycle(t *testing.T) {
	comp := fakeComponent{
		name:         "foo",
		dependencies: []string{"istiod", "cert-manager", "istiod"},
		enabled:      true,
	}
	registry := fakeRegistry{
		components: []Component{
			&comp,
			&fakeComponent{
				name:    "cert-manager",
				enabled: true,
			},
			&fakeComponent{
				name:    "istio",
				enabled: true,
			},
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContextWithRegistry(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, &registry, false))
	assert.False(t, ready)
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycles(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", dependencies: []string{"fake1", "directCycle"}, ready: true}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", dependencies: []string{"fake3"}, ready: true}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", dependencies: []string{"fake4"}, ready: true}
	nocycles := fakeComponent{name: "nocycles", dependencies: []string{"fake6", "fake5"}, ready: true}
	noDependencies := fakeComponent{name: "fake1", ready: true}

	registry := &fakeRegistry{
		components: []Component{
			&noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			&fakeComponent{name: "fake2", dependencies: []string{"indirectCycle1", "fake1"}, ready: true},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			&fakeComponent{name: "fake3", dependencies: []string{"fake2"}, ready: true},
			&fakeComponent{name: "fake4", dependencies: []string{"fake3"}, ready: true},
			&fakeComponent{name: "fake5", dependencies: []string{"fake1"}, ready: true},
			&fakeComponent{name: "fake6", dependencies: []string{"fake5"}, ready: true},
			&nocycles,
			&indirectCycle1,
			&indirectCycle2,
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	compContext := NewFakeContextWithRegistry(client, &vzapi.Verrazzano{}, registry, false)

	assert.False(t, ComponentDependenciesMet(&directCycle, compContext))
	assert.False(t, ComponentDependenciesMet(&indirectCycle1, compContext))
	assert.False(t, ComponentDependenciesMet(&indirectCycle2, compContext))
	assert.True(t, ComponentDependenciesMet(&nocycles, compContext))
	assert.True(t, ComponentDependenciesMet(&noDependencies, compContext))
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func Test_checkDependencies(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", dependencies: []string{"fake1", "directCycle"}, ready: true}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", dependencies: []string{"fake3"}, ready: true}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", dependencies: []string{"fake4"}, ready: true}
	nocycles := fakeComponent{name: "nocycles", dependencies: []string{"fake6", "fake5"}, ready: true}
	noDependencies := fakeComponent{name: "fake1", ready: true}

	registry := fakeRegistry{
		components: []Component{
			&noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			&fakeComponent{name: "fake2", dependencies: []string{"indirectCycle1", "fake1"}, ready: true},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			&fakeComponent{name: "fake3", dependencies: []string{"fake2"}, ready: true},
			&fakeComponent{name: "fake4", dependencies: []string{"fake3"}, ready: true},
			&fakeComponent{name: "fake5", dependencies: []string{"fake1"}, ready: true},
			&fakeComponent{name: "fake6", dependencies: []string{"fake5"}, ready: true},
			&nocycles,
			&indirectCycle1,
			&indirectCycle2,
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ctx := NewFakeContextWithRegistry(client, &vzapi.Verrazzano{}, &registry, false)

	_, err := CheckDependencies(&directCycle, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for directCycle")
	_, err = CheckDependencies(&indirectCycle1, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for indirectCycle1")
	_, err = CheckDependencies(&indirectCycle2, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for fake3")
	dependencies, err := CheckDependencies(&nocycles, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"fake6": true,
		"fake5": true,
		"fake1": true,
	}, dependencies)

	dependencies, err = CheckDependencies(&noDependencies, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{}, dependencies)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesChainNoCycle(t *testing.T) {
	chainNoCycle := fakeComponent{name: "chainNoCycle", dependencies: []string{"fake2"}, ready: true}
	repeatDepdendency := fakeComponent{name: "repeatDependency", dependencies: []string{"fake1", "fake2", "fake1"}, ready: true}
	registry := fakeRegistry{
		components: []Component{
			&fakeComponent{name: "fake1", ready: true},
			&fakeComponent{name: "fake2", dependencies: []string{"fake1"}, ready: true},
			&chainNoCycle,
			&repeatDepdendency,
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	compContext := NewFakeContextWithRegistry(client, &vzapi.Verrazzano{}, &registry, false)

	// Dependency chain, no cycle
	ready := ComponentDependenciesMet(&chainNoCycle, compContext)
	assert.True(t, ready)

	// Same dependency listed twice, not an error
	ready = ComponentDependenciesMet(&repeatDepdendency, compContext)
	assert.True(t, ready)
}

// TestNoComponentDependencies tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns true if there are no dependencies
func TestNoComponentDependencies(t *testing.T) {
	comp := fakeComponent{
		name:    "foo",
		enabled: true,
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(&comp, NewFakeContext(client, &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.True(t, ready)
}

type fakeComponent struct {
	name         string
	dependencies []string
	enabled      bool
	ready        bool
}

var _ Component = &fakeComponent{}

func (f *fakeComponent) Name() string {
	return f.name
}

func (f *fakeComponent) GetDependencies() []string {
	return f.dependencies
}

func (f *fakeComponent) IsReady(_ ComponentContext) bool {
	return f.ready
}

func (f *fakeComponent) IsEnabled(_ ComponentContext) bool {
	return f.enabled
}

func (f *fakeComponent) GetMinVerrazzanoVersion() string {
	return "1.0.0"
}

func (f *fakeComponent) IsOperatorInstallSupported() bool {
	return true
}

func (f *fakeComponent) IsInstalled(_ ComponentContext) (bool, error) {
	return true, nil
}

func (f *fakeComponent) PreInstall(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) Install(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) PostInstall(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) PreUpgrade(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) Upgrade(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) PostUpgrade(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) Reconcile(_ ComponentContext) error {
	return nil
}

func (f *fakeComponent) GetIngressNames(_ ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}
