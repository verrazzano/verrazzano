// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	assert.Len(comps, 17, "Wrong number of components")
	assert.Equal(comps[0].Name(), "istio-base")
	assert.Equal(comps[1].Name(), "istiod")
	assert.Equal(comps[2].Name(), "istio-ingress")
	assert.Equal(comps[3].Name(), "istio-egress")
	assert.Equal(comps[4].Name(), "istiocoredns")
	assert.Equal(comps[5].Name(), "ingress-controller")
	assert.Equal(comps[6].Name(), "cert-manager")
	assert.Equal(comps[7].Name(), "external-dns")
	assert.Equal(comps[8].Name(), "rancher")
	assert.Equal(comps[9].Name(), "verrazzano")
	assert.Equal(comps[10].Name(), "coherence-operator")
	assert.Equal(comps[11].Name(), "weblogic-operator")
	assert.Equal(comps[12].Name(), "oam-kubernetes-runtime")
	assert.Equal(comps[13].Name(), "verrazzano-application-operator")
	assert.Equal(comps[14].Name(), "mysql")
	assert.Equal(comps[15].Name(), "keycloak")
	assert.Equal(comps[16].Name(), "istio")
}

// TestFindComponent tests FindComponent
// GIVEN a component
//  WHEN I call FindComponent
//  THEN the true and the component are returned, false and an empty comp otherwise
func TestFindComponent(t *testing.T) {
	found, comp := FindComponent("istiod")
	assert.True(t, found)
	assert.NotNil(t, comp)
	assert.Equal(t, "istiod", comp.Name())
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
		Dependencies:    []string{"istiod"},
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
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
		Dependencies:    []string{"istiod"},
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
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
		Dependencies:    []string{"istiod"},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	//	ObjectMeta: metav1.ObjectMeta{
	//		Namespace: "istio-system",
	//		Name:      "istiod",
	//	},
	//	Status: appsv1.DeploymentStatus{
	//		Replicas:            1,
	//		ReadyReplicas:       0,
	//		AvailableReplicas:   0,
	//		UnavailableReplicas: 1,
	//	},
	//})
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusPendingInstall, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	ready := ComponentDependenciesMet(zap.S(), client, comp)
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
		Dependencies:    []string{"istiod", "cert-manager"},
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
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
		Dependencies:    []string{"istiod", "cert-manager"},
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
	assert.False(t, ready)
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
	ready := ComponentDependenciesMet(zap.S(), client, comp)
	assert.True(t, ready)
}
