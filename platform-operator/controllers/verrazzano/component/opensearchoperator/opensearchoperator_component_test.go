// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmconst "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	podTemplateHashLabel         = "pod-template-hash"
	dashboardLabelSelector       = "opensearch.cluster.dashboards"
	controllerManagerLabel       = "controller-manager"
	controlPlaneLabel            = "control-plane"

	profilesRelativePath = "../../../../manifests/profiles"
	esMaster             = "es-master"
	esData               = "es-data"
	esData1              = "es-data1"
	dataIngest           = "data-ingest"
	fakeDomain           = "mydomain.com"
)

// TestMonitorOverrides tests the MonitorOverrides function
// GIVEN a call to MonitorOverrides
// WHEN MonitorOverrides is called
// THEN false is returned
func TestMonitorOverrides(t *testing.T) {
	trueValue := true
	falseValue := false
	c := NewComponent()
	cli := fake.NewClientBuilder().WithScheme(testScheme).Build()
	tests := []struct {
		name   string
		vz     *v1alpha1.Verrazzano
		result bool
	}{
		{
			"OpenSearchOperator Component is nil",
			&v1alpha1.Verrazzano{},
			false,
		},
		{
			"OpenSearchOperator component initialised",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &v1alpha1.OpenSearchOperatorComponent{},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly enabled in OpenSearchOperator component",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &v1alpha1.OpenSearchOperatorComponent{
							InstallOverrides: v1alpha1.InstallOverrides{
								MonitorChanges: &trueValue},
						},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly disabled in OpenSearchOperator component",
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &v1alpha1.OpenSearchOperatorComponent{
							InstallOverrides: v1alpha1.InstallOverrides{
								MonitorChanges: &falseValue},
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

// TestGetDependencies tests the GetDependencies function
// GIVEN a call to GetDependencies
// WHEN GetDependencies is called
// THEN the OpenSearchOperator Component dependencies are returned
func TestGetDependencies(t *testing.T) {
	dependencies := NewComponent().GetDependencies()
	assert.Len(t, dependencies, 3)
	assert.Equal(t, networkpolicies.ComponentName, dependencies[0])
	assert.Equal(t, cmconst.ClusterIssuerComponentName, dependencies[1])
	assert.Equal(t, nginx.ComponentName, dependencies[2])
}

// TestOpenSearchOperatorEnabled tests if OpenSearchOperator is enabled
// GIVEN a call to IsEnabled
// WHEN the VZ CR is populated
// THEN a boolean is returned
func TestOpensearchOperatorEnabled(t *testing.T) {
	trueVal := true
	falseVal := false
	crA1 := &v1alpha1.Verrazzano{}
	crB1 := &v1beta1.Verrazzano{}

	crA1NilComp := crA1.DeepCopy()
	crA1NilComp.Spec.Components.OpenSearchOperator = nil
	crA1NilEnabled := crA1.DeepCopy()
	crA1NilEnabled.Spec.Components.OpenSearchOperator = &v1alpha1.OpenSearchOperatorComponent{Enabled: nil}
	crA1Enabled := crA1.DeepCopy()
	crA1Enabled.Spec.Components.OpenSearchOperator = &v1alpha1.OpenSearchOperatorComponent{Enabled: &trueVal}
	crA1Disabled := crA1.DeepCopy()
	crA1Disabled.Spec.Components.OpenSearchOperator = &v1alpha1.OpenSearchOperatorComponent{Enabled: &falseVal}

	crB1NilComp := crB1.DeepCopy()
	crB1NilComp.Spec.Components.OpenSearchOperator = nil
	crB1NilEnabled := crB1.DeepCopy()
	crB1NilEnabled.Spec.Components.OpenSearchOperator = &v1beta1.OpenSearchOperatorComponent{Enabled: nil}
	crB1Enabled := crB1.DeepCopy()
	crB1Enabled.Spec.Components.OpenSearchOperator = &v1beta1.OpenSearchOperatorComponent{Enabled: &trueVal}
	crB1Disabled := crB1.DeepCopy()
	crB1Disabled.Spec.Components.OpenSearchOperator = &v1beta1.OpenSearchOperatorComponent{Enabled: &falseVal}

	tests := []struct {
		name         string
		verrazzanoA1 *v1alpha1.Verrazzano
		verrazzanoB1 *v1beta1.Verrazzano
		assertion    func(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool
	}{
		{
			name:         "test v1alpha1 component nil",
			verrazzanoA1: crA1NilComp,
			assertion:    assert.True,
		},
		{
			name:         "test v1alpha1 enabled nil",
			verrazzanoA1: crA1NilEnabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1alpha1 enabled",
			verrazzanoA1: crA1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1alpha1 disabled",
			verrazzanoA1: crA1Disabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 component nil",
			verrazzanoB1: crB1NilComp,
			assertion:    assert.True,
		},
		{
			name:         "test v1beta1 enabled nil",
			verrazzanoB1: crB1NilEnabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1beta1 enabled",
			verrazzanoB1: crB1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1beta1 disabled",
			verrazzanoB1: crB1Disabled,
			assertion:    assert.False,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verrazzanoA1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCR()))
			}
			if tt.verrazzanoB1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCRV1Beta1()))
			}
		})
	}
}

// TestGetCertificateName tests the GetCertificateNames for the OpenSearchOperator component
func TestGetCertificateName(t *testing.T) {
	disabled := false

	scheme := k8scheme.Scheme

	var tests = []struct {
		name  string
		vz    v1alpha1.Verrazzano
		certs []types.NamespacedName
	}{
		// GIVEN a call to GetCertificate
		// WHEN OpenSearchOperator is disabled
		// THEN no certs are returned
		{
			name: "TestGetIngress when OpenSearchOperator disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &v1alpha1.OpenSearchOperatorComponent{Enabled: &disabled},
					},
				},
			},
		},
		// GIVEN a call to GetCertificate
		// WHEN OpenSearchOperator is enabled
		// THEN cluster certs are returned
		{
			name: "TestGetIngress when NGINX disabled",
			vz:   v1alpha1.Verrazzano{},
			certs: []types.NamespacedName{
				{Name: fmt.Sprintf("%s-admin-cert", clusterName), Namespace: ComponentNamespace},
				{Name: fmt.Sprintf("%s-dashboards-cert", clusterName), Namespace: ComponentNamespace},
				{Name: fmt.Sprintf("%s-master-cert", clusterName), Namespace: ComponentNamespace},
				{Name: fmt.Sprintf("%s-node-cert", clusterName), Namespace: ComponentNamespace},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			certs := NewComponent().GetCertificateNames(ctx)
			assert.Equal(t, test.certs, certs)
		})
	}
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
// WHEN the sts and deployments are ready
// THEN a true is returned
func TestIsReady(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	fakeClient := getReadyObjects().Build()
	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: "dev"}}, nil, false, profilesRelativePath)
	GetControllerRuntimeClient = func() (client.Client, error) {
		return fakeClient, nil
	}

	ready := NewComponent().IsReady(fakeCtx)
	assert.True(t, ready)
}

// TestIsNotReady tests the IsReady function
// GIVEN a call to IsReady
// WHEN the sts and deployments are not ready
// THEN a false is returned
func TestIsNotReady(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()
	fakeClient := getNotReadyObjects().Build()
	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: "dev"}}, nil, false, profilesRelativePath)
	GetControllerRuntimeClient = func() (client.Client, error) {
		return fakeClient, nil
	}
	assert.False(t, NewComponent().IsReady(fakeCtx))
}

// TestIsAvailable tests the IsAvailable function
// GIVEN a call to IsAvailable
// WHEN the sts and deployments are ready
// THEN a true is returned
func TestIsAvailable(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	fakeClient := getReadyObjects().Build()
	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: "dev"}}, nil, false, profilesRelativePath)
	GetControllerRuntimeClient = func() (client.Client, error) {
		return fakeClient, nil
	}

	_, availability := NewComponent().IsAvailable(fakeCtx)
	assert.Equal(t, v1alpha1.ComponentAvailability(v1alpha1.ComponentAvailable), availability)
}

// TestPreInstall tests the PreInstall function
// GIVEN a call to PreInstall
// WHEN OpenSearchOperator is pre-installed
// THEN no error is returned
func TestPreInstall(t *testing.T) {
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..",
	})

	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	fakeCtx := spi.NewFakeContext(fakeClient, getVZ(), nil, false, profilesRelativePath)
	GetControllerRuntimeClient = func() (client.Client, error) {
		return fakeClient, nil
	}
	err := NewComponent().PreInstall(fakeCtx)
	assert.NoError(t, err)
}

// TestPreInstall tests the PostInstall function
// GIVEN a call to PreInstall
// WHEN OpenSearchOperator is post-installed
// THEN no error is returned
func TestPostInstall(t *testing.T) {
	err := NewComponent().PostInstall(newFakeContext())
	assert.NoError(t, err)
}

func getNotReadyObjects() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      opensearchOperatorDeploymentName,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{controlPlaneLabel: controllerManagerLabel},
				}},
			Status: appsv1.DeploymentStatus{
				Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1, ReadyReplicas: 0},
		},
	)
}

func getReadyObjects() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      opensearchOperatorDeploymentName,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{controlPlaneLabel: controllerManagerLabel},
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
				Name:      opensearchOperatorDeploymentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					controlPlaneLabel:    controllerManagerLabel,
					podTemplateHashLabel: "95d8c5d96",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        opensearchOperatorDeploymentName + "-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
}
