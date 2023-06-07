// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	cmcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	cmconfig "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	cmocidns "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/webhookoci"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusteragent"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusterapi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusteroperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentbitosoutput"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"
	certManagerNamespace      = "cert-manager"
	profileDir                = "../../../../manifests/profiles"
)

var basicNoneClusterWithStatus = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-none",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "none",
	},
	Status: v1alpha1.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &v1alpha1.InstanceInfo{},
	},
}

var basicV1Beta1NoneClusterWithStatus = v1beta1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-none",
	},
	Spec: v1beta1.VerrazzanoSpec{
		Profile: "none",
	},
	Status: v1beta1.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &v1beta1.InstanceInfo{},
	},
}

// TestGetComponents tests getting the components
// GIVEN a component
//
//	WHEN I call GetComponents
//	THEN the Get returns the correct components
func TestGetComponents(t *testing.T) {
	a := assert.New(t)
	comps := GetComponents()

	var i int
	a.Len(comps, 40, "Wrong number of components")
	a.Equal(comps[i].Name(), networkpolicies.ComponentName)
	i++
	a.Equal(comps[i].Name(), fluentoperator.ComponentName)
	i++
	a.Equal(comps[i].Name(), fluentbitosoutput.ComponentName)
	i++
	a.Equal(comps[i].Name(), oam.ComponentName)
	i++
	a.Equal(comps[i].Name(), appoper.ComponentName)
	i++
	a.Equal(comps[i].Name(), istio.ComponentName)
	i++
	a.Equal(comps[i].Name(), weblogic.ComponentName)
	i++
	a.Equal(comps[i].Name(), nginx.ComponentName)
	i++
	a.Equal(comps[i].Name(), cmcontroller.ComponentName)
	i++
	a.Equal(comps[i].Name(), cmocidns.ComponentName)
	i++
	a.Equal(comps[i].Name(), cmconfig.ComponentName)
	i++
	a.Equal(comps[i].Name(), externaldns.ComponentName)
	i++
	a.Equal(comps[i].Name(), clusterapi.ComponentName)
	i++
	a.Equal(comps[i].Name(), rancher.ComponentName)
	i++
	a.Equal(comps[i].Name(), verrazzano.ComponentName)
	i++
	a.Equal(comps[i].Name(), vmo.ComponentName)
	i++
	a.Equal(comps[i].Name(), opensearch.ComponentName)
	i++
	a.Equal(comps[i].Name(), opensearchdashboards.ComponentName)
	i++
	a.Equal(comps[i].Name(), grafana.ComponentName)
	i++
	a.Equal(comps[i].Name(), grafanadashboards.ComponentName)
	i++
	a.Equal(comps[i].Name(), authproxy.ComponentName)
	i++
	a.Equal(comps[i].Name(), coherence.ComponentName)
	i++
	a.Equal(comps[i].Name(), mysqloperator.ComponentName)
	i++
	a.Equal(comps[i].Name(), mysql.ComponentName)
	i++
	a.Equal(comps[i].Name(), keycloak.ComponentName)
	i++
	a.Equal(comps[i].Name(), kiali.ComponentName)
	i++
	a.Equal(comps[i].Name(), promoperator.ComponentName)
	i++
	a.Equal(comps[i].Name(), promadapter.ComponentName)
	i++
	a.Equal(comps[i].Name(), kubestatemetrics.ComponentName)
	i++
	a.Equal(comps[i].Name(), pushgateway.ComponentName)
	i++
	a.Equal(comps[i].Name(), promnodeexporter.ComponentName)
	i++
	a.Equal(comps[i].Name(), jaegeroperator.ComponentName)
	i++
	a.Equal(comps[i].Name(), console.ComponentName)
	i++
	a.Equal(comps[i].Name(), fluentd.ComponentName)
	i++
	a.Equal(comps[i].Name(), velero.ComponentName)
	i++
	a.Equal(comps[i].Name(), rancherbackup.ComponentName)
	i++
	a.Equal(comps[i].Name(), clusteroperator.ComponentName)
	i++
	a.Equal(comps[i].Name(), argocd.ComponentName)
	i++
	a.Equal(comps[i].Name(), thanos.ComponentName)
	i++
	a.Equal(comps[i].Name(), clusteragent.ComponentName)
}

// TestFindComponent tests FindComponent
// GIVEN a component
//
//	WHEN I call FindComponent
//	THEN the true and the component are returned, false and an empty comp otherwise
func TestFindComponent(t *testing.T) {
	found, comp := FindComponent(istio.ComponentName)
	assert.True(t, found)
	assert.NotNil(t, comp)
	assert.Equal(t, istio.ComponentName, comp.Name())
}

// TestComponentDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN the true is returned if all dependencies are met
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}}, nil, true))
	assert.True(t, ready)
}

// TestComponentDependenciesNotMet tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN the false is returned if any dependencies are not met
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.False(t, ready)
}

// TestComponentOptionalDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN true is still returned if the dependency is not enabled
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
	assert.True(t, ready)
}

// TestComponentDependenciesDependencyChartNotInstalled tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN the false is returned if the dependent chart isn't installed
func TestComponentDependenciesDependencyChartNotInstalled(t *testing.T) {
	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{istio.ComponentName},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.False(t, ready)
}

// TestComponentMultipleDependenciesPartiallyMet tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN the false is returned if any depdencies are not met
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.False(t, ready)
}

func createRelease(name string, namespace string, releaseStatus release.Status, appVersion string) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: namespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        releaseStatus,
			Description:   "Named Release Stub",
		},
		Chart: &chart.Chart{Metadata: &chart.Metadata{
			AppVersion: appVersion,
		}},
		Version: 1,
	}
}

// TestComponentMultipleDependenciesMet tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN the true is returned if all dependencies are met
func TestComponentMultipleDependenciesMet(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})
	InitRegistry()
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

	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		actionConfig, err := helm.CreateActionConfig(false, certManagerDeploymentName, release.StatusDeployed, vzlog.DefaultLogger(), nil)
		if err != nil {
			return nil, err
		}
		if settings.Namespace() == vzconst.VerrazzanoSystemNamespace {
			err = actionConfig.Releases.Create(createRelease("oam-kubernetes-runtime", vzconst.VerrazzanoSystemNamespace, release.StatusDeployed, "0.3.0"))
			if err != nil {
				return nil, err
			}
		} else {
			err = actionConfig.Releases.Create(createRelease(certManagerDeploymentName, certManagerNamespace, release.StatusDeployed, "v1.9.1"))
			if err != nil {
				return nil, err
			}
			err = actionConfig.Releases.Create(createRelease(cainjectorDeploymentName, certManagerNamespace, release.StatusDeployed, "v1.9.1"))
			if err != nil {
				return nil, err
			}
			err = actionConfig.Releases.Create(createRelease(webhookDeploymentName, certManagerNamespace, release.StatusDeployed, "v1.9.1"))
			if err != nil {
				return nil, err
			}
		}

		return actionConfig, nil
	})

	comp := helm2.HelmComponent{
		ReleaseName:    "foo",
		ChartDir:       "chartDir",
		ChartNamespace: "bar",
		Dependencies:   []string{oam.ComponentName, cmcontroller.ComponentName},
	}
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.True(t, ready)
}

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN it returns false if there's a cycle in the dependencies
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
	ready := ComponentDependenciesMet(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: metav1.ObjectMeta{Namespace: "foo"}}, nil, false))
	assert.False(t, ready)
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN it returns false if there's a cycle in the dependencies
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
	assert.True(t, ComponentDependenciesMet(directDependency, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
	assert.False(t, ComponentDependenciesMet(dependent, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)))
}

// TestComponentDependenciesCycles tests ComponentDependenciesMet
// GIVEN a registry of components with dependencies, and some with cycles
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN it returns false if there's a cycle in the dependencies
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

// TestComponentDependenciesCycle tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN it returns false if there's a cycle in the dependencies
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

// TestComponentDependenciesCycles validates the test method checkDependencyCycles, which is used by another test to
// validate that the production component registry does not have any cycles declared.
//
// GIVEN a registry of components with dependencies, and some with cycles
//
//	WHEN I call checkDependencyCycles for it
//	THEN it returns an error if there's a cycle in the dependencies
func Test_checkDependencyCycles(t *testing.T) {
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

	_, err := checkDependencyCycles(directCycle, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for directCycle")
	_, err = checkDependencyCycles(indirectCycle1, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for indirectCycle1")
	_, err = checkDependencyCycles(indirectCycle2, ctx, make(map[string]bool), make(map[string]bool))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle found for fake3")
	dependencies, err := checkDependencyCycles(nocycles, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"fake6": true,
		"fake5": true,
		"fake1": true,
	}, dependencies)

	dependencies, err = checkDependencyCycles(noDependencies, ctx, make(map[string]bool), make(map[string]bool))
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{}, dependencies)
}

// TestRegistryDependencies tests the production Registry components for dependency cycles
// GIVEN a component
//
//	WHEN I call checkDependencies for it
//	THEN No error is returned that indicates a cycle in the chain
func TestRegistryDependencies(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	for _, comp := range GetComponents() {
		_, err := checkDependencyCycles(comp, spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false, profileDir),
			make(map[string]bool), make(map[string]bool))
		assert.NoError(t, err)
	}
}

// TestNoComponentDependencies tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN it returns true if there are no dependencies
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
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN returns true if a dependency's component status is already in Ready state
func TestComponentDependenciesMetStateCheckReady(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStateReady, true)
}

// TestComponentDependenciesMetStateCheckNotReady tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN returns false if a dependency's component status is not Ready state and the deployments are not ready
func TestComponentDependenciesMetStateCheckNotReady(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStatePreInstalling, true)
}

// TestComponentDependenciesMetStateCheckCompDisabled tests ComponentDependenciesMet
// GIVEN a component
//
//	WHEN I call ComponentDependenciesMet for it
//	THEN returns false if a dependency is disabled and the component status is disabled
func TestComponentDependenciesMetStateCheckCompDisabled(t *testing.T) {
	runDepenencyStateCheckTest(t, v1alpha1.CompStateDisabled, false)
}

// TestNoneProfileInstalledAllComponentsDisabled Tests the effectiveCR
// GIVEN when a verrazzano instance with NONE profile
// WHEN Newcontext is called
// THEN all components referred from the registry are disabled except for network-policies
func TestNoneProfileInstalledAllComponentsDisabled(t *testing.T) {
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1client.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset().ApiextensionsV1(), nil
	}

	config.TestProfilesDir = profileDir
	defer func() { config.TestProfilesDir = "" }()
	t.Run("TestNoneProfileInstalledAllComponentsDisabled", func(t *testing.T) {
		a := assert.New(t)
		log := vzlog.DefaultLogger()

		context, err := spi.NewContext(log, fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), &basicNoneClusterWithStatus, &basicV1Beta1NoneClusterWithStatus, false)
		context.EffectiveCRV1Beta1()
		assert.NoError(t, err)
		a.NotNil(context, "Context was nil")

		// Verify the v1alpha1 Actual and effective CR status
		a.NotNil(context.ActualCR(), "Actual CR was nil")
		a.Equal(basicNoneClusterWithStatus, *context.ActualCR(), "Actual CR unexpectedly modified")
		a.NotNil(context.EffectiveCR(), "Effective CR was nil")
		a.Equal(v1alpha1.VerrazzanoStatus{}, context.EffectiveCR().Status, "Effective CR status not empty")

		// Verify the v1beta1 Actual and effective CR status
		a.NotNil(context.ActualCRV1Beta1(), "Actual v1beta1 CR was nil")
		a.Equal(basicV1Beta1NoneClusterWithStatus, *context.ActualCRV1Beta1(), "Actual v1beta1 CR unexpectedly modified")
		a.NotNil(context.EffectiveCRV1Beta1(), "Effective v1beta1 CR was nil")
		a.Equal(v1beta1.VerrazzanoStatus{}, context.EffectiveCRV1Beta1().Status, "Effective v1beta1 CR status not empty")

		for _, comp := range GetComponents() {
			// Networkpolicies is expected to be installed always
			if comp.GetJSONName() == "verrazzanoNetworkPolicies" {
				continue
			}
			assert.False(t, comp.IsEnabled(context.EffectiveCR()), "Component %s not disabled in v1alpha1 \"none\" profile", comp.Name())
			assert.False(t, comp.IsEnabled(context.EffectiveCRV1Beta1()), "Component %s not disabled in v1beta1 \"none\" profile", comp.Name())
		}
	})
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

// checkDependencies Check the ready state of any dependencies and check for cycles
func checkDependencyCycles(c spi.Component, context spi.ComponentContext, visited map[string]bool, stateMap map[string]bool) (map[string]bool, error) {
	compName := c.Name()
	log := context.Log()
	log.Debugf("Checking %s dependencies", compName)
	if _, wasVisited := visited[compName]; wasVisited {
		return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, dependency cycle found for %s", c.Name())
	}
	visited[compName] = true
	for _, dependencyName := range c.GetDependencies() {
		if compName == dependencyName {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, dependency cycle found for %s", c.Name())
		}
		if _, ok := stateMap[dependencyName]; ok {
			// dependency already checked
			log.Debugf("Dependency %s already checked", dependencyName)
			continue
		}
		found, dependency := FindComponent(dependencyName)
		if !found {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, declared dependency not found for %s: %s", c.Name(), dependencyName)
		}
		if trace, err := checkDependencyCycles(dependency, context, visited, stateMap); err != nil {
			return trace, err
		}
		// Only check if dependency is ready when the dependency is enabled
		stateMap[dependencyName] = true
	}
	return stateMap, nil
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

func (f fakeComponent) IsAvailable(_ spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	var available v1alpha1.ComponentAvailability = v1alpha1.ComponentAvailable
	if !f.ready {
		available = v1alpha1.ComponentUnavailable
	}
	return "", available
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
