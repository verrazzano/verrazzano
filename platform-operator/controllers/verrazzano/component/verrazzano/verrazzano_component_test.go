// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestPreUpgrade tests the Verrazzano PreUpgrade call
// GIVEN a Verrazzano component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the TestFixupFluentdDaemonset unit tests, this just adds coverage
	// for the Component interface hook
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewFakeClientWithScheme(testScheme), &vzapi.Verrazzano{}, false))
	assert.NoError(t, err)
}

// TestIsReadySecretNotReady tests the Verrazzano IsReady call
// GIVEN a Verrazzano component
//  WHEN I call IsReady when it is installed and the deployment availability criteria are met, but the secret is not found
//  THEN false is returned
func TestIsReadySecretNotReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: globalconst.VerrazzanoSystemNamespace,
			Name:      "verrazzano-operator",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// TestIsReadyChartNotInstalled tests the Verrazzano IsReady call
// GIVEN a Verrazzano component
//  WHEN I call IsReady when it is not installed
//  THEN false is returned
func TestIsReadyChartNotInstalled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// TestIsReady tests the Verrazzano IsReady call
// GIVEN a Verrazzano component
//  WHEN I call IsReady when all requirements are met
//  THEN false is returned
func TestIsReady(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-authproxy",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-monitoring-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: globalconst.VerrazzanoSystemNamespace}},
	)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	assert.True(t, NewComponent().IsReady(ctx))
}

// TestIsReadyDeploymentNotAvailable tests the Verrazzano IsReady call
// GIVEN a Verrazzano component
//  WHEN I call IsReady when the VO deployment is not available
//  THEN false is returned
func TestIsReadyDeploymentNotAvailable(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-authproxy",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   0,
				UnavailableReplicas: 0,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-monitoring-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   0,
				UnavailableReplicas: 0,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-operator",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   0,
				UnavailableReplicas: 0,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: globalconst.VerrazzanoSystemNamespace}},
	)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// TestIsReadyDeploymentVMIDisabled tests the Verrazzano IsReady call
// GIVEN a Verrazzano component with all VMI components disabled
//  WHEN I call IsReady
//  THEN true is returned if only the verrazzano-authproxy is deployed
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-authproxy",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UnavailableReplicas: 0,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: globalconst.VerrazzanoSystemNamespace}},
	)
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Kibana:        &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Elasticsearch: &vzapi.ElasticsearchComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Prometheus:    &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Grafana:       &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
	}
	ctx := spi.NewFakeContext(client, vz, false)
	assert.True(t, NewComponent().IsReady(ctx))
}

// TestIsReadyDeploymentVMIDisabled tests the Verrazzano IsReady call
// GIVEN a Verrazzano component with all VMI components disabled
//  WHEN I call IsReady
//  THEN false is returned if only the verrazzano-authproxy is not available
func TestNotReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      "verrazzano-authproxy",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:            1,
				ReadyReplicas:       1,
				AvailableReplicas:   0,
				UnavailableReplicas: 0,
			},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: globalconst.VerrazzanoSystemNamespace}},
	)
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Kibana:        &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Elasticsearch: &vzapi.ElasticsearchComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Prometheus:    &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
		Grafana:       &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
	}
	ctx := spi.NewFakeContext(client, vz, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// TestPreInstall tests the Verrazzano PreInstall call
// GIVEN a Verrazzano component
//  WHEN I call PreInstall when dependencies are met
//  THEN no error is returned
func TestPreInstall(t *testing.T) {
	client := createPreInstallTestClient()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	err := NewComponent().PreInstall(ctx)
	assert.NoError(t, err)
}

// TestPostInstall tests the Verrazzano PostInstall call
// GIVEN a Verrazzano component
//  WHEN I call PostInstall
//  THEN no error is returned
func TestPostInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	err := NewComponent().PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the Verrazzano PostUpgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN a Verrazzano component upgrading from 1.1.0 to 1.2.0
//  WHEN I call PostUpgrade
//  THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Version: "v1.2.0"},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"}}, false)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}

func createPreInstallTestClient(extraObjs ...runtime.Object) client.Client {
	objs := []runtime.Object{}
	objs = append(objs, extraObjs...)
	client := fake.NewFakeClientWithScheme(testScheme, objs...)
	return client
}
