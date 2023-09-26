// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	_ = vmov1.AddToScheme(testScheme)
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
	k8sclient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(k8sclient, nil, nil, false)
	assert.NoError(t, NewComponent().PreInstall(ctx))
	vmoCRD := &apiextensionsv1.CustomResourceDefinition{}
	// The VMO CRD should exist after PreInstall
	assert.NoError(t, k8sclient.Get(context.TODO(), types.NamespacedName{Name: "verrazzanomonitoringinstances.verrazzano.io"}, vmoCRD))
}

// TestPreUpgrade tests the VMO PreUpgrade call
// GIVEN a VMO component
//
//	WHEN I call PreUpgrade with defaults
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helm.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time.Now()
			return &release.Release{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = testHelmConfigDir
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), nil, nil, false))
	assert.NoError(t, err)
}

// TestPreUninstall tests the PreUninstall for VMO Component
// GIVEN  a VMO component, existing system VMI resource and NO VMO labeled deployments/statefulsets/ingresses
// WHEN I call PreUninstall
// THEN the VMI is deleted, and no error is returned
// GIVEN a VMO component, existing system VMI resource and VMO labeled deployments/statefulsets/ingresses
// WHEN I call PreUninstall
// THEN the VMI is deleted, and an error is returned
func TestPreUninstall(t *testing.T) {
	testVMI := vmov1.VerrazzanoMonitoringInstance{
		ObjectMeta: metav1.ObjectMeta{Name: common.VMIName, Namespace: constants.VerrazzanoSystemNamespace},
	}
	vmoLabels := map[string]string{vmoconst.VMOLabel: common.VMIName}
	vmoDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-vmo-deploy",
			Labels:    vmoLabels,
		},
	}
	vmoIngress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-vmo-ing",
			Labels:    vmoLabels,
		},
	}
	vmoStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-vmo-sts",
			Labels:    vmoLabels,
		},
	}
	vmoSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-vmo-svc",
			Labels:    vmoLabels,
		},
	}
	nonVMODeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-deploy",
		},
	}
	nonVMOIngress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      "test-ing",
		},
	}
	tests := []struct {
		name      string
		vmi       *vmov1.VerrazzanoMonitoringInstance
		otherObjs []client.Object
		expectErr bool
	}{
		{
			name:      "Test PreUninstall when no VMI and no VMO objects exist",
			vmi:       nil,
			otherObjs: nil,
			expectErr: false,
		},
		{
			name:      "Test PreUninstall when only VMI exists",
			vmi:       &testVMI,
			otherObjs: nil,
			expectErr: false,
		},
		{
			name:      "Test PreUninstall when VMI and a VMO deployment exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&vmoDeploy},
			expectErr: true,
		},
		{
			name:      "Test PreUninstall when VMI and a VMO ingress exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&vmoIngress},
			expectErr: true,
		},
		{
			name:      "Test PreUninstall when VMI and a VMO statefulset exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&vmoStatefulSet},
			expectErr: true,
		},
		{
			name:      "Test PreUninstall when VMI and a VMO service exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&vmoSvc},
			expectErr: true,
		},
		{
			name:      "Test PreUninstall when VMI and multiple VMO-labeled resources exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&vmoDeploy, &vmoIngress, &vmoStatefulSet, &vmoSvc},
			expectErr: true,
		},
		{
			name:      "Test PreUninstall when VMI and a non-VMO labeled deployment/ingress exist",
			vmi:       &testVMI,
			otherObjs: []client.Object{&nonVMODeploy, &nonVMOIngress},
			expectErr: false,
		},
	}
	config.TestHelmConfigDir = testHelmConfigDir
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientBldr := fake.NewClientBuilder().WithScheme(testScheme)
			if tt.vmi != nil {
				clientBldr = clientBldr.WithObjects(tt.vmi)
			}
			if tt.otherObjs != nil {
				clientBldr = clientBldr.WithObjects(tt.otherObjs...)
			}
			k8sclient := clientBldr.Build()
			ctx := spi.NewFakeContext(k8sclient, nil, nil, false)
			err := NewComponent().PreUninstall(ctx)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			// The system VMI should have been deleted after PreUninstall
			vmi := vmov1.VerrazzanoMonitoringInstance{}
			assert.True(t, errors.IsNotFound(k8sclient.Get(context.TODO(),
				types.NamespacedName{Name: common.VMIName, Namespace: constants.VerrazzanoSystemNamespace},
				&vmi)))
		})
	}
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
	return helm.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), createRelease)
}

func testActionConfigWithoutInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), nil)
}

// TestUninstallHelmChartInstalled tests the VMO Uninstall call
// GIVEN a VMO component
//
//	WHEN I call Uninstall with the VMO helm chart installed
//	THEN no error is returned
func TestUninstallHelmChartInstalled(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallation)
	k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace(constants.VerrazzanoSystemNamespace)
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
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
	k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace(constants.VerrazzanoSystemNamespace)
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}
