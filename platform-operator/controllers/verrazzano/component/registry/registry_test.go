// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"
	certManagerNamespace      = "cert-manager"
	profileDir                = "../../../../manifests/profiles"
)

// TestGetComponents tests getting the components
// GIVEN a component
//  WHEN I call GetComponents
//  THEN the Get returns the correct components
func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetComponents()

	assert.Len(comps, 13, "Wrong number of components")
	assert.Equal(comps[0].Name(), nginx.ComponentName)
	assert.Equal(comps[1].Name(), certmanager.ComponentName)
	assert.Equal(comps[2].Name(), externaldns.ComponentName)
	assert.Equal(comps[3].Name(), rancher.ComponentName)
	assert.Equal(comps[4].Name(), verrazzano.ComponentName)
	assert.Equal(comps[5].Name(), coherence.ComponentName)
	assert.Equal(comps[6].Name(), weblogic.ComponentName)
	assert.Equal(comps[7].Name(), oam.ComponentName)
	assert.Equal(comps[8].Name(), appoper.ComponentName)
	assert.Equal(comps[9].Name(), mysql.ComponentName)
	assert.Equal(comps[10].Name(), keycloak.ComponentName)
	assert.Equal(comps[11].Name(), kiali.ComponentName)
	assert.Equal(comps[12].Name(), istio.ComponentName)
}

// TestFindComponent tests FindComponent
// GIVEN a component
//  WHEN I call FindComponent
//  THEN the true and the component are returned, false and an empty comp otherwise
func TestFindComponent(t *testing.T) {
	found, comp := FindComponent(istio.ComponentName)
	assert.True(t, found)
	assert.NotNil(t, comp)
	assert.Equal(t, istio.ComponentName, comp.Name())
}

// TestComponentDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the true is returned if all depdencies are met
func TestComponentDependenciesMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{istio.ComponentName},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      "istiod",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}, false))
	assert.True(t, ready)
}

// TestComponentDependenciesNotMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if any depdencies are not met
func TestComponentDependenciesNotMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{istio.ComponentName},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      "istiod",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.False(t, ready)
}

// TestComponentDependenciesDependencyChartNotInstalled tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if the dependent chart isn't installed
func TestComponentDependenciesDependencyChartNotInstalled(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{istio.ComponentName},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusPendingInstall, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.False(t, ready)
}

// TestComponentMultipleDependenciesPartiallyMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if any depdencies are not met
func TestComponentMultipleDependenciesPartiallyMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{istio.ComponentName, "cert-manager"},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      "istiod",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.False(t, ready)
}

// TestComponentMultipleDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the true is returned if all depdencies are met
func TestComponentMultipleDependenciesMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{istio.ComponentName, "cert-manager"},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, newReadyDeployment("istiod", "istio-system"),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace))
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.True(t, ready)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycle(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:     "foo",
		ChartDir:        "chartDir",
		ChartNamespace:  "bar",
		ReadyStatusFunc: nil,
		Dependencies:    []string{"istiod", "cert-manager", "istiod"},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newReadyDeployment("istiod", "istio-system"),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace))
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.False(t, ready)
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycles(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", dependencies: []string{"fake1", "directCycle"}}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", dependencies: []string{"fake3"}}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", dependencies: []string{"fake4"}}
	nocycles := fakeComponent{name: "nocycles", dependencies: []string{"fake6", "fake5"}}
	noDependencies := fakeComponent{name: "fake1"}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			fakeComponent{name: "fake2", dependencies: []string{"indirectCycle1", "fake1"}},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			fakeComponent{name: "fake3", dependencies: []string{"fake2"}},
			fakeComponent{name: "fake4", dependencies: []string{"fake3"}},
			fakeComponent{name: "fake5", dependencies: []string{"fake1"}},
			fakeComponent{name: "fake6", dependencies: []string{"fake5"}},
			nocycles,
			indirectCycle1,
			indirectCycle2,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	assert.False(t, ComponentDependenciesMet(directCycle, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)))
	assert.False(t, ComponentDependenciesMet(indirectCycle1, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)))
	assert.False(t, ComponentDependenciesMet(indirectCycle2, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)))
	assert.True(t, ComponentDependenciesMet(nocycles, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)))
	assert.True(t, ComponentDependenciesMet(noDependencies, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)))
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func Test_checkDependencies(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", dependencies: []string{"fake1", "directCycle"}}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", dependencies: []string{"fake3"}}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", dependencies: []string{"fake4"}}
	nocycles := fakeComponent{name: "nocycles", dependencies: []string{"fake6", "fake5"}}
	noDependencies := fakeComponent{name: "fake1"}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			fakeComponent{name: "fake2", dependencies: []string{"indirectCycle1", "fake1"}},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			fakeComponent{name: "fake3", dependencies: []string{"fake2"}},
			fakeComponent{name: "fake4", dependencies: []string{"fake3"}},
			fakeComponent{name: "fake5", dependencies: []string{"fake1"}},
			fakeComponent{name: "fake6", dependencies: []string{"fake5"}},
			nocycles,
			indirectCycle1,
			indirectCycle2,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false)

	_, err := checkDependencies(directCycle, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for directCycle")
	_, err = checkDependencies(indirectCycle1, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for indirectCycle1")
	_, err = checkDependencies(indirectCycle2, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for fake3")
	dependencies, err := checkDependencies(nocycles, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"fake6": true,
		"fake5": true,
		"fake1": true,
	}, dependencies)

	dependencies, err = checkDependencies(noDependencies, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{}, dependencies)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesChainNoCycle(t *testing.T) {
	chainNoCycle := fakeComponent{name: "chainNoCycle", dependencies: []string{"fake2"}}
	repeatDepdendency := fakeComponent{name: "repeatDependency", dependencies: []string{"fake1", "fake2", "fake1"}}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{name: "fake1"},
			fakeComponent{name: "fake2", dependencies: []string{"fake1"}},
			chainNoCycle,
			repeatDepdendency,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// Dependency chain, no cycle
	ready := ComponentDependenciesMet(chainNoCycle, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false))
	assert.True(t, ready)

	// Same dependency listed twice, not an error
	ready = ComponentDependenciesMet(repeatDepdendency, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false))
	assert.True(t, ready)
}

// TestRegistryDependencies tests the default Registry components for cycles
// GIVEN a component
//  WHEN I call checkDependencies for it
//  THEN No error is returned that indicates a cycle in the chain
func TestRegistryDependencies(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	for _, comp := range GetComponents() {
		_, err := checkDependencies(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, false, profileDir),
			make(map[string]bool), make(map[string]bool))
		assert.NoError(t, err)
	}
}

// TestNoComponentDependencies tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns true if there are no dependencies
func TestNoComponentDependencies(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.True(t, ready)
}

// Create a new deployment object for testing
func newReadyDeployment(name string, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	}
}

type fakeComponent struct {
	name         string
	dependencies []string
	enabled      bool
}

var _ spi.Component = fakeComponent{}

func (f fakeComponent) Name() string {
	return f.name
}

func (f fakeComponent) GetDependencies() []string {
	return f.dependencies
}

func (f fakeComponent) IsReady(_ spi.ComponentContext) bool {
	return true
}

func (f fakeComponent) IsEnabled(_ spi.ComponentContext) bool {
	return f.enabled
}

func (f fakeComponent) GetMinVerrazzanoVersion() string {
	return "1.0.0"
}

func (f fakeComponent) IsOperatorInstallSupported() bool {
	return true
}

func (f fakeComponent) IsInstalled(_ spi.ComponentContext) (bool, error) {
	return true, nil
}

func (f fakeComponent) PreInstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Install(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Upgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Reconcile(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}
