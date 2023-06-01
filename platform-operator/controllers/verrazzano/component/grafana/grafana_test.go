// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFilePath = "../../testdata/test_bom.json"

var testScheme = runtime.NewScheme()
var replicas int32

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = vmov1.AddToScheme(testScheme)
}

// TestIsGrafanaInstalled tests the isGrafanaInstalled function for the Grafana component
func TestIsGrafanaInstalled(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Grafana deployment exists
			// WHEN we call isGrafanaInstalled
			// THEN the call returns true
			name: "Test isGrafanaInstalled when Grafana is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Grafana deployment does not exist
			// WHEN we call isGrafanaInstalled
			// THEN the call returns false
			name:       "Test isGrafanaInstalled when Grafana deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isGrafanaInstalled(ctx))
		})
	}
}

// TestIsGrafanaReady tests the isGrafanaReady function
func TestIsGrafanaReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Grafana deployment exists and there are available replicas
			// AND the Grafana admin secret exists
			// WHEN we call isGrafanaReady
			// THEN the call returns true
			name: "Test isGrafanaReady when Grafana is successfully deployed and the admin secret exists",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
						Labels:    map[string]string{"app": "system-grafana"},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "system-grafana"},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"app":               "system-grafana",
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        grafanaDeployment + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.GrafanaSecret,
						Namespace: ComponentNamespace,
					},
					Data: map[string][]byte{},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Grafana deployment exists and there are available replicas
			// AND the Grafana admin secret does not exist
			// WHEN we call isGrafanaReady
			// THEN the call returns false
			name: "Test isGrafanaReady when Grafana is successfully deployed and the admin secret does not exist",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Grafana deployment exists and there are no available replicas
			// WHEN we call isGrafanaReady
			// THEN the call returns false
			name: "Test isGrafanaReady when Grafana is deployed but there are no available replicas",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      grafanaDeployment,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
			).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, isGrafanaReady(ctx))
			ctx = spi.NewFakeContext(tt.client, &vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Components: vzapi.ComponentSpec{Grafana: &vzapi.GrafanaComponent{Replicas: &replicas}}}}, nil, false)
			assert.Equal(t, true, isGrafanaReady(ctx))
		})
	}
}

// testActionConfigNoInstallation fakes a Helm release that is in the uninstalled state
func testActionConfigNoInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, "my-release", release.StatusUninstalled, vzlog.DefaultLogger(), nil)
}

// structs that allow us to unmarshal the Grafana datasources configmap YAML
type datasource struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

type datasources struct {
	Datasources []datasource `json:"datasources"`
}

// TestApplyDatasourcesConfigmap tests the applyDatasourcesConfigmap function.
func TestApplyDatasourcesConfigmap(t *testing.T) {
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..",
	})

	config.SetDefaultBomFilePath(testBomFilePath)

	// needed to fake the Helm calls otherwise Helm tries to connect to a running cluster
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigNoInstallation)

	trueValue := true
	falseValue := false
	thanosEnabledCR := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Thanos: &vzapi.ThanosComponent{
					Enabled: &trueValue,
				},
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	thanosAndPrometheusDisabledCR := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{
					Enabled: &falseValue,
				},
				PrometheusOperator: &vzapi.PrometheusOperatorComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	tests := []struct {
		name                   string
		vzCR                   *vzapi.Verrazzano
		expectPromDatasource   bool
		expectThanosDatasource bool
	}{
		// GIVEN Thanos is disabled and Prometheus is enabled
		// WHEN the applyDatasourcesConfigmap function is called
		// THEN the Grafana datasources configmap contains Prometheus as the only datasource
		{
			name:                   "Thanos is disabled (by default)",
			vzCR:                   &vzapi.Verrazzano{},
			expectPromDatasource:   true,
			expectThanosDatasource: false,
		},
		// GIVEN Thanos is enabled and Prometheus is enabled
		// WHEN the applyDatasourcesConfigmap function is called
		// THEN the Grafana datasources configmap contains Thanos and Prometheus datasources and Thanos is the default
		{
			name:                   "Thanos is enabled, Query Frontend is enabled by default",
			vzCR:                   thanosEnabledCR,
			expectPromDatasource:   true,
			expectThanosDatasource: true,
		},
		// GIVEN Thanos and Prometheus are both disabled
		// WHEN the applyDatasourcesConfigmap function is called
		// THEN the Grafana datasources configmap contains no datasources
		{
			name:                   "Thanos and Prometheus both disabled",
			vzCR:                   thanosAndPrometheusDisabledCR,
			expectPromDatasource:   false,
			expectThanosDatasource: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			ctx := spi.NewFakeContext(client, tt.vzCR, nil, false)
			err := applyDatasourcesConfigmap(ctx)
			assert.NoError(t, err)

			cm := &v1.ConfigMap{}
			err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: datasourcesConfigMapName}, cm)
			assert.NoError(t, err)
			dsYaml := cm.Data["datasource.yaml"]
			ds := &datasources{}
			err = yaml.Unmarshal([]byte(dsYaml), ds)
			assert.NoError(t, err)

			promIsDefault := false
			thanosIsDefault := false
			for _, d := range ds.Datasources {
				if d.Name == "Prometheus" {
					assert.True(t, tt.expectPromDatasource, "Found an unexpected Prometheus datasource")
					promIsDefault = d.IsDefault
				} else if d.Name == "Thanos" {
					assert.True(t, tt.expectThanosDatasource, "Found an unexpected Thanos datasource")
					thanosIsDefault = d.IsDefault
				} else {
					assert.Fail(t, "Found unexpected datasource name", "Name", d.Name)
				}
			}

			// if we expect Thanos in the list of datasources, we also expect it to be the default datasource, otherwise Prometheus should
			// be marked as the default datasource if it's enabled
			assert.Equal(t, tt.expectThanosDatasource, thanosIsDefault)
			assert.Equal(t, !tt.expectThanosDatasource && tt.expectPromDatasource, promIsDefault)
		})
	}
}

// TestIsThanosQueryFrontendEnabled tests the isThanosQueryFrontendEnabled function.
func TestIsThanosQueryFrontendEnabled(t *testing.T) {
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..",
	})

	config.SetDefaultBomFilePath(testBomFilePath)

	// needed to fake the Helm calls otherwise Helm tries to connect to a running cluster
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigNoInstallation)

	client := fake.NewClientBuilder().Build()

	trueValue := true
	falseValue := false
	thanosEnabledCR := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Thanos: &vzapi.ThanosComponent{
					Enabled: &trueValue,
				},
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	thanosQueryFrontendDisabledCR := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Thanos: &vzapi.ThanosComponent{
					Enabled: &trueValue,
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(`{"queryFrontend": {"enabled": false}}`),
								},
							},
						},
					},
				},
				Ingress: &vzapi.IngressNginxComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	tests := []struct {
		name          string
		ctx           spi.ComponentContext
		expectEnabled bool
	}{
		// GIVEN Thanos is disabled
		// WHEN the isThanosQueryFrontendEnabled function is called
		// THEN the function should return false
		{
			name:          "Thanos is disabled (by default)",
			ctx:           spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false),
			expectEnabled: false,
		},
		// GIVEN Thanos is enabled
		// WHEN the isThanosQueryFrontendEnabled function is called
		// THEN the function should return true
		{
			name:          "Thanos is enabled, Query Frontend is enabled by default",
			ctx:           spi.NewFakeContext(client, thanosEnabledCR, nil, false),
			expectEnabled: true,
		},
		// GIVEN Thanos is enabled but the Query Frontend is disabled
		// WHEN the isThanosQueryFrontendEnabled function is called
		// THEN the function should return false
		{
			name:          "Thanos is enabled, Query Frontend is explicitly disabled",
			ctx:           spi.NewFakeContext(client, thanosQueryFrontendDisabledCR, nil, false),
			expectEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabled, err := isThanosQueryFrontendEnabled(tt.ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectEnabled, enabled)
		})
	}
}

// TestRestartGrafanaPod tests the restartGrafanaPod function.
func TestRestartGrafanaPod(t *testing.T) {
	// GIVEN no Grafana deployment exists
	// WHEN the restartGrafanaPod function is called
	// THEN a NotFound error is returned
	client := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := restartGrafanaPod(ctx)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// GIVEN the Grafana deployment exists
	// WHEN the restartGrafanaPod function is called
	// THEN no error is returned and the deployment template annotation has been added
	client = fake.NewClientBuilder().WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      grafanaDeployment,
			},
		},
	).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err = restartGrafanaPod(ctx)
	assert.NoError(t, err)

	deployment := &appsv1.Deployment{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: grafanaDeployment}, deployment)
	assert.NoError(t, err)
	assert.Contains(t, deployment.Spec.Template.ObjectMeta.Annotations, vzconst.VerrazzanoRestartAnnotation)
}
