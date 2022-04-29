// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	vmoDeployment = "verrazzano-monitoring-operator"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vmov1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = vzclusters.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestIsReadySecretNotReady tests the OpenSearch-Dashboards areOpenSearchDashboardsReady call
// GIVEN an OpenSearch-Dashboards component
//  WHEN I call areOpenSearchDashboardsReady when it is installed and the deployment availability criteria are met, but the secret is not found
//  THEN false is returned
func TestIsReadySecretNotReady(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Console:       &vzapi.ConsoleComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Grafana:       &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      vmoDeployment,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}).Build()
	ctx := spi.NewFakeContext(c, vz, false)
	assert.False(t, checkOpenSearchDashboardsStatus(ctx, status.DeploymentsAreReady))
}

// TestIsReadyNotInstalled tests the OpenSearch-Dashboards areOpenSearchDashboardsReady call
// GIVEN an OpenSearch-Dashboards component
//  WHEN I call areOpenSearchDashboardsReady when it is not installed
//  THEN false is returned
func TestIsReadyNotInstalled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, false)
	assert.False(t, checkOpenSearchDashboardsStatus(ctx, status.DeploymentsAreReady))
}

// TestIsReady tests the areOpenSearchDashboardsReady call
// GIVEN OpenSearch-Dashboards components that are all enabled by default
//  WHEN I call areOpenSearchDashboardsReady when all requirements are met
//  THEN false is returned
func TestIsReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      kibanaDeployment,
				Labels:    map[string]string{"app": "system-kibana"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
	).Build()

	vz := &vzapi.Verrazzano{}
	ctx := spi.NewFakeContext(c, vz, false)
	assert.True(t, checkOpenSearchDashboardsStatus(ctx, status.DeploymentsAreReady))
}

// TestIsReadyDeploymentNotAvailable tests the OpenSearch-Dashboards areOpenSearchDashboardsReady call
// GIVEN an OpenSearch-Dashboards component
//  WHEN I call areOpenSearchDashboardsReady when the Kibana deployment is not available
//  THEN false is returned
func TestIsReadyDeploymentNotAvailable(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      kibanaDeployment,
				Labels:    map[string]string{"app": "system-kibana"},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   0,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
	).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, false)
	assert.False(t, checkOpenSearchDashboardsStatus(ctx, status.DeploymentsAreReady))
}

// TestIsReadyDeploymentVMIDisabled tests the OpenSearch-Dashboards areOpenSearchDashboardsReady call
// GIVEN an OpenSearch-Dashboards component with all VMI components disabled
//  WHEN I call areOpenSearchDashboardsReady
//  THEN true is returned
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
		Namespace: ComponentNamespace}},
	).Build()
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Console:       &vzapi.ConsoleComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Grafana:       &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
	}
	ctx := spi.NewFakeContext(c, vz, false)
	assert.True(t, checkOpenSearchDashboardsStatus(ctx, status.DeploymentsAreReady))
}
