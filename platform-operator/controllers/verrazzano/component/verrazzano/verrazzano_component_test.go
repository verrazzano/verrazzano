// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"testing"

	"github.com/stretchr/testify/assert"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

var dnsComponents = vzapi.ComponentSpec{
	DNS: &vzapi.DNSComponent{
		External: &vzapi.External{Suffix: "blah"},
	},
}

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Verrazzano: &vzapi.VerrazzanoComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

func readyOpenSearchPods() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod",
			Namespace: globalconst.VerrazzanoSystemNamespace,
			Labels: map[string]string{
				"app": workloadName,
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  containerName,
					Ready: true,
				},
			},
		},
	}
// fakeUpgrade override the upgrade function during unit tests
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

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
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "verrazzano",
				Namespace: globalconst.VerrazzanoSystemNamespace,
			},
		},
		readyOpenSearchPods(),
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
//  THEN true is returned
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
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
//  THEN true is returned
func TestNotReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(testScheme,
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

// TestInstall tests the Verrazzano Install call
// GIVEN a Verrazzano component
//  WHEN I call Install when dependencies are met
//  THEN no error is returned
func TestInstall(t *testing.T) {
	client := createPreInstallTestClient()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	helmcli.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helmcli.ChartStatusDeployed, nil
	})
	defer helmcli.SetDefaultChartStateFunction()
	err := NewComponent().Install(ctx)
	assert.NoError(t, err)
}

// TestPostInstall tests the Verrazzano PostInstall call
// GIVEN a Verrazzano component
//  WHEN I call PostInstall
//  THEN no error is returned
func TestPostInstall(t *testing.T) {
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = `{"status": 404}`
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	client := fake.NewFakeClientWithScheme(testScheme, readyOpenSearchPods())
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, false)
	vzComp := NewComponent()

	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := vzComp.(verrazzanoComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		client.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	err = vzComp.PostInstall(ctx)
	assert.NoError(t, err)
}

// TestUpgrade tests the Verrazzano Upgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN a Verrazzano component upgrading from 1.1.0 to 1.2.0
//  WHEN I call Upgrade
//  THEN no error is returned
func TestUpgrade(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version:    "v1.2.0",
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	err := NewComponent().Upgrade(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the Verrazzano PostUpgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN a Verrazzano component upgrading from 1.1.0 to 1.2.0
//  WHEN I call PostUpgrade
//  THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme, readyOpenSearchPods())
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = `{"status": 404}`
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version:    "v1.2.0",
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, false)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}

func createPreInstallTestClient(extraObjs ...runtime.Object) client.Client {
	objs := []runtime.Object{}
	objs = append(objs, extraObjs...)
	client := fake.NewFakeClientWithScheme(testScheme, objs...)
	return client
}

// TestIsEnabledNilVerrazzano tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Verrazzano component is nil
//  THEN true is returned
func TestIsEnabledNilVerrazzano(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Verrazzano = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Verrazzano component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath)))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Verrazzano component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Verrazzano.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Verrazzano component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Verrazzano.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Verrazzano component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Verrazzano.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

func getBoolPtr(b bool) *bool {
	return &b
}
