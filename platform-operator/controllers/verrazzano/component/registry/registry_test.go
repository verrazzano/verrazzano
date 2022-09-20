// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancherbackup"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	a.Len(comps, 29, "Wrong number of components")
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
	a.Equal(comps[15].Name(), mysqloperator.ComponentName)
	a.Equal(comps[16].Name(), mysql.ComponentName)
	a.Equal(comps[17].Name(), keycloak.ComponentName)
	a.Equal(comps[18].Name(), kiali.ComponentName)
	a.Equal(comps[19].Name(), promoperator.ComponentName)
	a.Equal(comps[20].Name(), promadapter.ComponentName)
	a.Equal(comps[21].Name(), kubestatemetrics.ComponentName)
	a.Equal(comps[22].Name(), pushgateway.ComponentName)
	a.Equal(comps[23].Name(), promnodeexporter.ComponentName)
	a.Equal(comps[24].Name(), jaegeroperator.ComponentName)
	a.Equal(comps[25].Name(), console.ComponentName)
	a.Equal(comps[26].Name(), fluentd.ComponentName)
	a.Equal(comps[27].Name(), velero.ComponentName)
	a.Equal(comps[28].Name(), rancherbackup.ComponentName)
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
				Labels:    map[string]string{"control-plane": "coherence"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"control-plane": "coherence"},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconst.VerrazzanoSystemNamespace,
				Name:      "coherence-operator" + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"control-plane":     "coherence",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   vzconst.VerrazzanoSystemNamespace,
				Name:        "coherence-operator" + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}, nil, true))
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{
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
	}, nil, false))
	assert.False(t, ready)
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
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
		newReadyDeployment("oam-kubernetes-runtime", vzconst.VerrazzanoSystemNamespace, map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"}),
		newPod("oam-kubernetes-runtime", vzconst.VerrazzanoSystemNamespace, map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"}),
		newReplicaSet("oam-kubernetes-runtime", vzconst.VerrazzanoSystemNamespace),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace, map[string]string{"app": certManagerDeploymentName}),
		newPod(certManagerDeploymentName, certManagerNamespace, map[string]string{"app": certManagerDeploymentName}),
		newReplicaSet(certManagerDeploymentName, certManagerNamespace),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace, map[string]string{"app": "cainjector"}),
		newPod(cainjectorDeploymentName, certManagerNamespace, map[string]string{"app": "cainjector"}),
		newReplicaSet(cainjectorDeploymentName, certManagerNamespace),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace, map[string]string{"app": "webhook"}),
		newPod(webhookDeploymentName, certManagerNamespace, map[string]string{"app": "webhook"}),
		newReplicaSet(webhookDeploymentName, certManagerNamespace),
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

	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
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
		newReadyDeployment("istiod", "istio-system", nil),
		newReadyDeployment(certManagerDeploymentName, certManagerNamespace, nil),
		newReadyDeployment(cainjectorDeploymentName, certManagerNamespace, nil),
		newReadyDeployment(webhookDeploymentName, certManagerNamespace, nil)).Build()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.False(t, ready)
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestIndirectDependencyMetButNotReady(t *testing.T) {
	// directCycle -> fake1, directCycle
	indirectDependency := fakeComponent{name: "indirectDependency", enabled: false, ready: true}
	directDependency := fakeComponent{name: "directDependency", enabled: true, ready: false, dependencies: []string{"indirectDependency"}}
	dependent := fakeComponent{name: "dependent", enabled: false, ready: false, dependencies: []string{"directDependency"}}

	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			directDependency,
			indirectDependency,
			dependent,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	assert.True(t, ComponentDependenciesMet(indirectDependency, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.False(t, ComponentDependenciesMet(directDependency, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.False(t, ComponentDependenciesMet(dependent, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func TestComponentDependenciesCycles(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", enabled: true, dependencies: []string{"fake1", "directCycle"}}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", enabled: true, dependencies: []string{"fake3"}}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", enabled: true, dependencies: []string{"fake4"}}
	nocycles := fakeComponent{name: "nocycles", enabled: true, ready: true, dependencies: []string{"fake6", "fake5"}}
	noDependencies := fakeComponent{name: "fake1", enabled: true, ready: true}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			fakeComponent{name: "fake2", enabled: true, dependencies: []string{"indirectCycle1", "fake1"}},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			fakeComponent{name: "fake3", enabled: true, dependencies: []string{"fake2"}},
			fakeComponent{name: "fake4", enabled: true, dependencies: []string{"fake3"}},
			fakeComponent{name: "fake5", enabled: true, ready: true, dependencies: []string{"fake1"}},
			fakeComponent{name: "fake6", enabled: true, ready: true, dependencies: []string{"fake5"}},
			nocycles,
			indirectCycle1,
			indirectCycle2,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	assert.False(t, ComponentDependenciesMet(directCycle, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.False(t, ComponentDependenciesMet(indirectCycle1, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.False(t, ComponentDependenciesMet(indirectCycle2, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.True(t, ComponentDependenciesMet(nocycles, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.True(t, ComponentDependenciesMet(noDependencies, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//  WHEN I call ComponentDependenciesMet for it
//  THEN it returns false if there's a cycle in the dependencies
func Test_checkDependencies(t *testing.T) {
	// directCycle -> fake1, directCycle
	directCycle := fakeComponent{name: "directCycle", enabled: true, dependencies: []string{"fake1", "directCycle"}}
	// indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
	indirectCycle1 := fakeComponent{name: "indirectCycle1", enabled: true, dependencies: []string{"fake3"}}
	// indirectCycle2 -> fake4 -> fake3 -> fake2 -> indirectCycle -> fake3
	indirectCycle2 := fakeComponent{name: "indirectCycle2", enabled: true, dependencies: []string{"fake4"}}
	nocycles := fakeComponent{name: "nocycles", enabled: true, dependencies: []string{"fake6", "fake5"}}
	noDependencies := fakeComponent{name: "fake1", enabled: true, ready: true}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			noDependencies,
			// fake2 -> indirectCycle1 -> fake3 -> fake2 -> indirectCycle1
			fakeComponent{name: "fake2", enabled: true, dependencies: []string{"indirectCycle1", "fake1"}},
			// fake3 -> fake2 -> indirectCycle1 -> fake3
			fakeComponent{name: "fake3", enabled: true, dependencies: []string{"fake2"}},
			fakeComponent{name: "fake4", enabled: true, dependencies: []string{"fake3"}},
			fakeComponent{name: "fake5", enabled: true, ready: true, dependencies: []string{"fake1"}},
			fakeComponent{name: "fake6", enabled: true, ready: true, dependencies: []string{"fake5"}},
			nocycles,
			indirectCycle1,
			indirectCycle2,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)

	_, err := checkDependencies(directCycle, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for directCycle")
	//_, err = checkDependencies(indirectCycle1, ctx, make(map[string]bool), make(map[string]bool))
	//assert.Error(t, err)
	//assert.Contains(t, err.Error(), "dependency cycle found for indirectCycle1")
	//_, err = checkDependencies(indirectCycle2, ctx, make(map[string]bool), make(map[string]bool))
	//assert.Error(t, err)
	//assert.Contains(t, err.Error(), "dependency cycle found for fake3")
	dependencies, err := checkDependencies(nocycles, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"fake6": true,
		"fake5": true,
		//"fake1": true,
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
	chainNoCycle := fakeComponent{name: "chainNoCycle", enabled: true, ready: true, dependencies: []string{"fake2"}}
	repeatDepdendency := fakeComponent{name: "repeatDependency", enabled: true, ready: true, dependencies: []string{"fake1", "fake2", "fake1"}}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{name: "fake1", enabled: true, ready: true},
			fakeComponent{name: "fake2", enabled: true, ready: true, dependencies: []string{"fake1"}},
			chainNoCycle,
			repeatDepdendency,
		}
	})
	defer ResetGetComponentsFn()

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Dependency chain, no cycle
	ready := ComponentDependenciesMet(chainNoCycle, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false))
	assert.True(t, ready)

	// Same dependency listed twice, not an error
	ready = ComponentDependenciesMet(repeatDepdendency, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false))
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
		_, err := checkDependencies(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false, profileDir),
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.True(t, ready)
}

// TestComponentDependenciesMetStateCheckReady tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN returns true if a dependency's component status is already in Ready state
func TestComponentDependenciesMetStateCheckReady(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStateReady, true)
}

// TestComponentDependenciesMetStateCheckNotReady tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN returns false if a dependency's component status is not Ready state and the deployments are not ready
func TestComponentDependenciesMetStateCheckNotReady(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStatePreInstalling, true)
}

// TestComponentDependenciesMetStateCheckCompDisabled tests ComponentDependenciesMet
// GIVEN a component
//  WHEN I call ComponentDependenciesMet for it
//  THEN returns false if a dependency is disabled and the component status is disabled
func TestComponentDependenciesMetStateCheckCompDisabled(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStateDisabled, false)
}

func runDepenencyStateCheckTest(t *testing.T, state v1alpha1.CompStateType, enabled bool) {
	const compName = coherence.ComponentName
	comp := fakeComponent{name: "foo", enabled: true, dependencies: []string{compName}}

	dependency := fakeComponent{name: compName, enabled: true, ready: false}
	OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			dependency,
		}
	})
	defer ResetGetComponentsFn()

	expectedResult := true
	if state != v1alpha1.CompStateReady {
		expectedResult = false
	}

	cr := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CoherenceOperator: &v1alpha1.CoherenceOperatorComponent{
					Enabled: &enabled,
				},
			},
		},
		Status: v1alpha1.VerrazzanoStatus{
			Components: v1alpha1.ComponentStatusMap{
				compName: &v1alpha1.ComponentStatusDetails{
					Name:  compName,
					State: state,
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, cr, nil, true))
	assert.Equal(t, expectedResult, ready)
}

// Create a new deployment object for testing
func newReadyDeployment(name string, namespace string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}
}

func newPod(name string, namespace string, labelsIn map[string]string) *corev1.Pod {
	lablels := make(map[string]string)
	lablels["pod-template-hash"] = "95d8c5d96"
	for key, element := range labelsIn {
		lablels[key] = element
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels:    lablels,
		},
	}
	return pod
}

func newReplicaSet(name string, namespace string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

type fakeComponent struct {
	name         string
	namespace    string
	dependencies []string
	enabled      bool
	ready        bool
}

var _ spi.Component = fakeComponent{}

func (f fakeComponent) Name() string {
	return f.name
}

func (f fakeComponent) Namespace() string {
	return f.namespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done
func (f fakeComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

func (f fakeComponent) GetJSONName() string {
	return f.name
}

func (f fakeComponent) GetOverrides(_ runtime.Object) interface{} {
	return []v1alpha1.Overrides{}
}

func (f fakeComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

func (f fakeComponent) GetDependencies() []string {
	return f.dependencies
}

func (f fakeComponent) IsReady(_ spi.ComponentContext) bool {
	return f.ready
}

func (f fakeComponent) IsEnabled(_ runtime.Object) bool {
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

func (f fakeComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return nil
}

func (f fakeComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	return nil
}

func (f fakeComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}
