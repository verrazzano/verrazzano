// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"testing"

	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	promadapter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/adapter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/kubestatemetrics"
	promnodeexporter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/nodeexporter"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/pushgateway"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
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
	a := assert.New(t)
	comps := GetComponents()

	a.Len(comps, 27, "Wrong number of components")
	a.Equal(comps[0].Name(), oam.ComponentName)
	a.Equal(comps[1].Name(), appoper.ComponentName)
	a.Equal(comps[2].Name(), istio.ComponentName)
	a.Equal(comps[3].Name(), weblogic.ComponentName)
	a.Equal(comps[4].Name(), nginx.ComponentName)
	a.Equal(comps[5].Name(), certmanager.ComponentName)
	a.Equal(comps[6].Name(), externaldns.ComponentName)
	a.Equal(comps[7].Name(), rancher.ComponentName)
	a.Equal(comps[8].Name(), verrazzano.ComponentName)
	a.Equal(comps[9].Name(), vmo.ComponentName)
	a.Equal(comps[10].Name(), opensearch.ComponentName)
	a.Equal(comps[11].Name(), opensearchdashboards.ComponentName)
	a.Equal(comps[12].Name(), grafana.ComponentName)
	a.Equal(comps[13].Name(), authproxy.ComponentName)
	a.Equal(comps[14].Name(), coherence.ComponentName)
	a.Equal(comps[15].Name(), mysql.ComponentName)
	a.Equal(comps[16].Name(), keycloak.ComponentName)
	a.Equal(comps[17].Name(), kiali.ComponentName)
	a.Equal(comps[18].Name(), promoperator.ComponentName)
	a.Equal(comps[19].Name(), promadapter.ComponentName)
	a.Equal(comps[20].Name(), kubestatemetrics.ComponentName)
	a.Equal(comps[21].Name(), pushgateway.ComponentName)
	a.Equal(comps[22].Name(), promnodeexporter.ComponentName)
	a.Equal(comps[23].Name(), jaegeroperator.ComponentName)
	a.Equal(comps[24].Name(), console.ComponentName)
	a.Equal(comps[25].Name(), fluentd.ComponentName)
	a.Equal(comps[26].Name(), velero.ComponentName)
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
//  THEN the true is returned if all dependencies are met
func TestComponentDependenciesMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{coherence.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconst.VerrazzanoSystemNamespace,
				Name:      "coherence-operator",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}, true))
	assert.True(t, ready)
}

// TestComponentDependenciesNotMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if any dependencies are not met
func TestComponentDependenciesNotMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{istio.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      "istiod",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.False(t, ready)
}

// TestComponentOptionalDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN true is still returned if the dependency is not enabled
func TestComponentOptionalDependenciesMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{istio.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	enabled := false
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client,
		&v1alpha1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "foo",
			},
			Spec: v1alpha1.VerrazzanoSpec{
				Components: v1alpha1.ComponentSpec{
					Istio: &v1alpha1.IstioComponent{
						Enabled: &enabled,
					},
				},
			},
		},
		false))
	assert.True(t, ready)
}

// TestComponentDependenciesDependencyChartNotInstalled tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN the false is returned if the dependent chart isn't installed
func TestComponentDependenciesDependencyChartNotInstalled(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{istio.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
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
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{istio.ComponentName, "cert-manager"},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      "istiod",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
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
//  THEN the true is returned if all dependencies are met
func TestComponentMultipleDependenciesMet(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{oam.ComponentName, certmanager.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newReadyDeployment("oam-kubernetes-runtime", vzconst.VerrazzanoSystemNamespace),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace),
	).Build()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	helm.SetChartInfoFunction(func(chartDir string) (helm.ChartInfo, error) {
		return helm.ChartInfo{
			AppVersion: "1.0",
		}, nil
	})
	defer helm.SetDefaultChartInfoFunction()

	helm.SetReleaseAppVersionFunction(func(releaseName string, namespace string) (string, error) {
		return "1.0", nil
	})
	defer helm.SetDefaultReleaseAppVersionFunction()

	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, false))
	assert.True(t, ready)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycle(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{"istiod", "cert-manager", "istiod"},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newReadyDeployment("istiod", "istio-system"),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace)).Build()
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

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
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

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
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

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

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
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

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
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
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
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
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

func (f fakeComponent) GetJSONName() string {
	return f.name
}

func (f fakeComponent) GetOverrides(_ *v1alpha1.Verrazzano) []v1alpha1.Overrides {
	return []v1alpha1.Overrides{}
}

func (f fakeComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

func (f fakeComponent) GetDependencies() []string {
	return f.dependencies
}

func (f fakeComponent) IsReady(_ spi.ComponentContext) bool {
	return true
}

func (f fakeComponent) IsEnabled(effectiveCR *v1alpha1.Verrazzano) bool {
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

func (f fakeComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (f fakeComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Uninstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostUninstall(_ spi.ComponentContext) error {
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

func (f fakeComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	return nil
}

func (f fakeComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	return nil
}

func (f fakeComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}
