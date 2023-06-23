// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	bootstrapOcneProvider     = "bootstrap-ocne"
	controlPlaneOcneProvider  = "control-plane-ocne"
	clusterAPIProvider        = "cluster-api"
	infrastructureOciProvider = "infrastructure-oci"

	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	podTemplateHashLabel         = "pod-template-hash"
	providerLabel                = "cluster.x-k8s.io/provider"
)

func fakeClusterAPINew(_ string, _ ...client.Option) (client.Client, error) {
	return &FakeClusterAPIClient{}, nil
}

// TestNewComponent tests the NewComponent function
// GIVEN a call to NewComponent
//
//	WHEN NewComponent is called
//	THEN a ClusterAPI Component is returned
func TestNewComponent(t *testing.T) {
	comp := NewComponent()
	assert.Empty(t, comp)
}

// TestName tests the Name function
// GIVEN a call to Name
//
//	WHEN Name is called
//	THEN the ClusterAPI Component name is returned
func TestName(t *testing.T) {
	var comp clusterAPIComponent
	name := comp.Name()
	assert.Equal(t, ComponentName, name)
}

// TestNamespace tests the Namespace function
// GIVEN a call to Namespace
//
//	WHEN Namespace is called
//	THEN the ClusterAPI Component namespace is returned
func TestNamespace(t *testing.T) {
	var comp clusterAPIComponent
	namespace := comp.Namespace()
	assert.Equal(t, ComponentNamespace, namespace)
}

// TestShouldInstallBeforeUpgrade tests the ShouldInstallBeforeUpgrade function
// GIVEN a call to ShouldInstallBeforeUpgrade
//
//	WHEN ShouldInstallBeforeUpgrade is called
//	THEN true is returned
func TestShouldInstallBeforeUpgrade(t *testing.T) {
	var comp clusterAPIComponent
	flag := comp.ShouldInstallBeforeUpgrade()
	assert.Equal(t, true, flag)
}

// TestGetDependencies tests the GetDependencies function
// GIVEN a call to GetDependencies
//
//	WHEN GetDependencies is called
//	THEN the ClusterAPI Component dependencies are returned
func TestGetDependencies(t *testing.T) {
	var comp clusterAPIComponent
	dependencies := comp.GetDependencies()
	assert.Len(t, dependencies, 2)
	assert.Equal(t, cmconstants.CertManagerComponentName, dependencies[0])
	assert.Equal(t, cmconstants.ClusterIssuerComponentName, dependencies[1])
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsReady(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.True(t, comp.IsReady(compContext))
}

// TestIsNotReady tests the IsReady function
// GIVEN a call to IsReady
//
//	WHEN the deployment object does not have enough replicas available
//	THEN false is returned
func TestIsNotReady(t *testing.T) {
	fakeClient := getNotReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.False(t, comp.IsReady(compContext))
}

// TestIsAvailable tests the IsAvailable function
// GIVEN a call to IsAvailable
//
//	WHEN deployments are available
//	THEN true is returned
func TestIsAvailable(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	reason, _ := comp.IsAvailable(compContext)
	assert.Equal(t, "", reason)
}

// TestIsNotAvailable tests the IsAvailable function
// GIVEN a call to IsAvailable
//
//	WHEN deployments are not available
//	THEN false is returned
func TestIsNotAvailable(t *testing.T) {
	fakeClient := getNotReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	reason, _ := comp.IsAvailable(compContext)
	assert.Equal(t, "deployment verrazzano-capi/capi-controller-manager not available: 0/1 replicas ready", reason)
}

// TestIsEnabled verifies ClusterAPI is enabled or disabled as expected
// GIVEN a Verrzzano CR
//
//	WHEN IsEnabled is called
//	THEN IsEnabled should return true/false depending on the enabled state of the CR
func TestIsEnabled(t *testing.T) {
	enabled := true
	disabled := false
	c := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
	vzWithClusterAPI := v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				ClusterAPI: &v1alpha1.ClusterAPIComponent{
					Enabled: &enabled,
				},
			},
		},
	}
	vzNoClusterAPI := v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				ClusterAPI: &v1alpha1.ClusterAPIComponent{
					Enabled: &disabled,
				},
			},
		},
	}
	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		enabled  bool
	}{
		{
			"should be enabled",
			spi.NewFakeContext(c, &vzWithClusterAPI, nil, false),
			true,
		},
		{
			"should not be enabled",
			spi.NewFakeContext(c, &vzNoClusterAPI, nil, false),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			r := NewComponent()
			assert.Equal(t, tt.enabled, r.IsEnabled(tt.ctx.EffectiveCR()))
		})
	}
}

// TestGetMinVerrazzanoVersion tests the GetMinVerrazzanoVersion function
// GIVEN a call to GetMinVerrazzanoVersion
//
//	WHEN GetMinVerrazzanoVersion is called
//	THEN the ClusterAPI Component minimum version is returned
func TestGetMinVerrazzanoVersion(t *testing.T) {
	var comp clusterAPIComponent
	version := comp.GetMinVerrazzanoVersion()
	assert.Equal(t, constants.VerrazzanoVersion1_6_0, version)
}

// TestGetIngressNames tests the GetIngressNames function
// GIVEN a call to GetIngressNames
//
//	WHEN GetIngressNames is called
//	THEN the no ClusterAPI ingresses are returned
func TestGetIngressNames(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	ingresses := comp.GetIngressNames(compContext)
	assert.Len(t, ingresses, 0)
}

// TestGetCertificateNames tests the GetCertificateNames function
// GIVEN a call to GetCertificateNames
//
//	WHEN GetCertificateNames is called
//	THEN the no ClusterAPI certificates are returned
func TestGetCertificateNames(t *testing.T) {
	fakeClient := getReadyDeployments().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	certificates := comp.GetCertificateNames(compContext)
	assert.Len(t, certificates, 0)
}

// TestGetJSONName tests the GetJSONName function
// GIVEN a call to GetJSONName
//
//	WHEN GetJSONName is called
//	THEN the ClusterAPI JSON name is returned
func TestGetJSONName(t *testing.T) {
	var comp clusterAPIComponent
	json := comp.GetJSONName()
	assert.Equal(t, ComponentJSONName, json)
}

// TestMonitorOverrides tests the MonitorOverrides function
// GIVEN a call to MonitorOverrides
//
//	WHEN MonitorOverrides is called
//	THEN false is returned
func TestMonitorOverrides(t *testing.T) {
	c := NewComponent()
	cli := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	tests := []struct {
		name   string
		vz     *v1alpha1.Verrazzano
		result bool
	}{
		{
			"ClusterAPI Component is nil",
			&v1alpha1.Verrazzano{},
			false,
		},
		{
			"ClusterAPI component initialised",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterAPI: &v1alpha1.ClusterAPIComponent{},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly enabled in ClusterAPI component",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterAPI: &v1alpha1.ClusterAPIComponent{
							InstallOverrides: v1alpha1.InstallOverrides{
								MonitorChanges: getBoolPtr(true)},
						},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly disabled in ClusterAPI component",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterAPI: &v1alpha1.ClusterAPIComponent{
							InstallOverrides: v1alpha1.InstallOverrides{
								MonitorChanges: getBoolPtr(false)},
						},
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(cli, tt.vz, nil, false)
			result := c.MonitorOverrides(ctx)
			assert.Equal(t, tt.result, result)
		})
	}
}

func getBoolPtr(b bool) *bool {
	return &b
}

// TestIsOperatorInstallSupported tests the IsOperatorInstallSupported function
// GIVEN a call to IsOperatorInstallSupported
//
//	WHEN IsOperatorInstallSupported is called
//	THEN true is returned
func TestIsOperatorInstallSupported(t *testing.T) {
	var comp clusterAPIComponent
	install := comp.IsOperatorInstallSupported()
	assert.Equal(t, true, install)
}

// TestIsOperatorUninstallSupported tests the IsOperatorUninstallSupported function
// GIVEN a call to IsOperatorUninstallSupported
//
//	WHEN IsOperatorUninstallSupported is called
//	THEN true is returned
func TestIsOperatorUninstallSupported(t *testing.T) {
	var comp clusterAPIComponent
	uninstall := comp.IsOperatorUninstallSupported()
	assert.Equal(t, true, uninstall)
}

// TestIsInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN ClusterAPI is installed
//	THEN true is returned
func TestIsInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiCMDeployment,
			},
		}).Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	installed, err := comp.IsInstalled(compContext)
	assert.NoError(t, err)
	assert.True(t, installed)
}

// TestIsNotInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//
//	WHEN ClusterAPI is not installed
//	THEN false is returned
func TestIsNotInstalled(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	installed, err := comp.IsInstalled(compContext)
	assert.NoError(t, err)
	assert.False(t, installed)
}

// TestPreInstall tests the PreInstall function
// GIVEN a call to PreInstall
//
//	WHEN ClusterAPI is pre-installed
//	THEN no error is returned
func TestPreInstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	config.TestHelmConfigDir = TestHelmConfigDir
	dir := os.TempDir() + "/" + time.Now().Format("20060102150405")
	setClusterAPIDir(dir)
	defer resetClusterAPIDir()
	defer os.RemoveAll(dir)
	var comp clusterAPIComponent
	err := comp.PreInstall(compContext)
	assert.NoError(t, err)
}

// TestInstall tests the Install function
// GIVEN a call to Install
//
//	WHEN ClusterAPI is installed
//	THEN no error is returned
func TestInstall(t *testing.T) {
	SetCAPIInitFunc(fakeClusterAPINew)
	defer ResetCAPIInitFunc()
	config.SetDefaultBomFilePath(testBomFilePath)
	config.TestHelmConfigDir = TestHelmConfigDir

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	err := comp.Install(compContext)
	assert.NoError(t, err)
}

// TestUninstall tests the Uninstall function
// GIVEN a call to Uninstall
//
//	WHEN ClusterAPI is Uninstalled
//	THEN no error is returned
func TestUninstall(t *testing.T) {
	SetCAPIInitFunc(fakeClusterAPINew)
	defer ResetCAPIInitFunc()
	config.SetDefaultBomFilePath(testBomFilePath)

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	err := comp.Uninstall(compContext)
	assert.NoError(t, err)
}

// TestPreUpgrade tests the PreUpgrade function
// GIVEN a call to PreUpgrade
//
//	WHEN ClusterAPI is pre-upgraded
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	config.TestHelmConfigDir = TestHelmConfigDir
	dir := os.TempDir() + "/" + time.Now().Format("20060102150405")
	setClusterAPIDir(dir)
	defer resetClusterAPIDir()
	defer os.RemoveAll(dir)
	var comp clusterAPIComponent
	err := comp.PreUpgrade(compContext)
	assert.NoError(t, err)
}

// WHEN ClusterAPI is upgraded
// THEN no error is returned
func TestUpgrade(t *testing.T) {
	SetCAPIInitFunc(fakeClusterAPINew)
	defer ResetCAPIInitFunc()
	config.SetDefaultBomFilePath(testBomFilePath)

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	var comp clusterAPIComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	err := comp.Upgrade(compContext)
	assert.NoError(t, err)
}

// TestValidateUpdate tests webhook updates
// GIVEN a call to ValidateUpdate
//
//	WHEN the ClusterAPI component is updated in a vz v1alpha1 resource
//	THEN expected result is returned
func TestValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1alpha1.Verrazzano
		new     *v1alpha1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterAPI: &v1alpha1.ClusterAPIComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1alpha1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterAPI: &v1alpha1.ClusterAPIComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &v1alpha1.Verrazzano{},
			new:     &v1alpha1.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateUpdateV1Beta1 tests webhook updates
// GIVEN a call to ValidateUpdateV1Beta1
//
//	WHEN the ClusterAPI component is updated in a vz v1beta1 resource
//	THEN expected result is returned
func TestValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						ClusterAPI: &v1beta1.ClusterAPIComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						ClusterAPI: &v1beta1.ClusterAPIComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &v1beta1.Verrazzano{},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdateV1Beta1(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getNotReadyDeployments() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiCMDeployment,
				Labels:    map[string]string{providerLabel: clusterAPIProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: clusterAPIProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiCMDeployment + "-95d8c5d97-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d97",
					providerLabel:        clusterAPIProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        capiCMDeployment + "-95d8c5d97",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
}

func getReadyDeployments() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiCMDeployment,
				Labels:    map[string]string{providerLabel: clusterAPIProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: clusterAPIProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiCMDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					providerLabel:        clusterAPIProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        capiCMDeployment + "-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiOcneBootstrapCMDeployment,
				Labels:    map[string]string{providerLabel: bootstrapOcneProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: bootstrapOcneProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiOcneBootstrapCMDeployment + "-95d8c5d93-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d93",
					providerLabel:        bootstrapOcneProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        capiOcneBootstrapCMDeployment + "-95d8c5d93",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiOcneControlPlaneCMDeployment,
				Labels:    map[string]string{providerLabel: controlPlaneOcneProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: controlPlaneOcneProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiOcneControlPlaneCMDeployment + "-95d8c5d92-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d92",
					providerLabel:        controlPlaneOcneProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        capiOcneControlPlaneCMDeployment + "-95d8c5d92",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},

		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiociCMDeployment,
				Labels:    map[string]string{providerLabel: infrastructureOciProvider},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{providerLabel: infrastructureOciProvider},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      capiociCMDeployment + "-95d8c5d91-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d91",
					providerLabel:        infrastructureOciProvider,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        capiociCMDeployment + "-95d8c5d91",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
}
