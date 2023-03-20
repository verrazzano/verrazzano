// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const profilesRelativePath = "../../../../manifests/profiles"
const testHelmConfigDir = "../../../../helm_config"

var testScheme *runtime.Scheme

func init() {
	testScheme = runtime.NewScheme()
	_ = rbacv1.AddToScheme(testScheme)
	_ = corev1.AddToScheme(testScheme)
	_ = netv1.AddToScheme(testScheme)
	_ = appsv1.AddToScheme(testScheme)
	_ = apiextensionsv1.AddToScheme(testScheme)
}

// TestIsEnabled tests the VMO IsEnabled call
// GIVEN a VMO component
//
//	WHEN I call IsEnabled
//	THEN true or false is returned
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			name: "Test IsEnabled when all VMI component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &falseValue,
						},
						Kibana: &vzapi.KibanaComponent{
							Enabled: &falseValue,
						},
						Grafana: &vzapi.GrafanaComponent{
							Enabled: &falseValue,
						},
						Prometheus: &vzapi.PrometheusComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false, profilesRelativePath)
			if tt.expectTrue {
				assert.True(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			} else {
				assert.False(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			}
		})
	}
}

// TestIsInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN the deployment object is found
//	THEN true is returned
func TestIsInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
	}).Build()
	installed, err := NewComponent().IsInstalled(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	assert.True(t, installed)
}

// TestIsNotInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN the deployment object is not found
//	THEN false is returned
func TestIsNotInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	installed, err := NewComponent().IsInstalled(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
	assert.False(t, installed)
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"k8s-app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"k8s-app": ComponentName},
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
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"k8s-app":           ComponentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	assert.True(t, NewComponent().IsReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, true)))
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the VMO is not ready per Helm
//	THEN true is returned
func TestIsNotReady(t *testing.T) {
	assert.False(t, NewComponent().IsReady(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false)))
}

// TestPostUpgrade tests the VMO PostUpgrade call
// GIVEN a VMO component
//
//	WHEN I call PostUpgrade with defaults
//	THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	// The actual post-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	err := NewComponent().PostUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), nil, nil, false))
	assert.NoError(t, err)
}

func TestPreInstall(t *testing.T) {
	config.TestHelmConfigDir = testHelmConfigDir
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	assert.NoError(t, NewComponent().PreInstall(ctx))
	vmoCRD := &apiextensionsv1.CustomResourceDefinition{}
	// The VMO CRD should exist after PreInstall
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: "verrazzanomonitoringinstances.verrazzano.io"}, vmoCRD))
}

// TestPreUpgrade tests the VMO PreUpgrade call
// GIVEN a VMO component
//
//	WHEN I call PreUpgrade with defaults
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStateFunction()

	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = testHelmConfigDir
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), nil, nil, false))
	assert.NoError(t, err)
}

func createRelease(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Version: 1,
	}
}

func testActionConfigWithInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(true, ComponentName, release.StatusDeployed, createRelease, vzlog.DefaultLogger())
}

func testActionConfigWithoutInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, ComponentName, release.StatusDeployed, nil, vzlog.DefaultLogger())
}

// TestUninstallHelmChartInstalled tests the VMO Uninstall call
// GIVEN a VMO component
//
//	WHEN I call Uninstall with the VMO helm chart installed
//	THEN no error is returned
func TestUninstallHelmChartInstalled(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallation)
	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestUninstallHelmChartNotInstalled tests the VMO Uninstall call
// GIVEN a VMO component
//
//	WHEN I call Uninstall with the VMO helm chart not installed
//	THEN no error is returned
func TestUninstallHelmChartNotInstalled(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithoutInstallation)

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}
