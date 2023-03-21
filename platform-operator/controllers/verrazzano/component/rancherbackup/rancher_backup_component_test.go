// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	profilesRelativePath = "../../../../manifests/profiles"
	validOverrideJSON    = "{\"serviceAccount\": {\"create\": false}}"
)

var enabled = true
var rancherBackupEnabledCR = &v1alpha1.Verrazzano{
	Spec: v1alpha1.VerrazzanoSpec{
		Components: v1alpha1.ComponentSpec{
			RancherBackup: &v1alpha1.RancherBackupComponent{
				Enabled: &enabled,
			},
		},
	},
}

// TestIsEnabled tests the IsEnabled function for the Rancher Backup Operator component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   v1alpha1.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Rancher Backup Operator component
			// THEN the call returns false
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   v1alpha1.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Rancher Backup Operator enabled
			// WHEN we call IsReady on the Rancher Backup Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when Rancher Backup Operator component set to enabled",
			actualCR:   *rancherBackupEnabledCR,
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Rancher Backup Operator disabled
			// WHEN we call IsReady on the Rancher Backup Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Rancher Backup Operator component set to disabled",
			actualCR: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						RancherBackup: &v1alpha1.RancherBackupComponent{
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
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// TestIsInstalled verifies component IsInstalled checks presence of the
// Rancher Backup operator deployment
func TestIsInstalled(t *testing.T) {
	var tests = []struct {
		name        string
		client      crtclient.Client
		isInstalled bool
	}{
		{
			"installed when Rancher Backup deployment is present",
			fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			).Build(),
			true,
		},
		{
			"not installed when Rancher Backup deployment is absent",
			fake.NewClientBuilder().WithScheme(testScheme).Build(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, rancherBackupEnabledCR, nil, false)
			installed, err := NewComponent().IsInstalled(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.isInstalled, installed)
		})
	}
}

func testActionConfigWithoutInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), nil)
}

func TestInstallUpgrade(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})
	v := NewComponent()

	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithoutInstallation)
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(rancherBackupEnabledCR).Build()
	ctx := spi.NewFakeContext(client, rancherBackupEnabledCR, nil, false)
	err := v.Install(ctx)
	assert.NoError(t, err)
	err = v.Upgrade(ctx)
	assert.NoError(t, err)
	err = v.Reconcile(ctx)
	assert.NoError(t, err)
}

// GIVEN a verrazzano CR with enabled rancher backup
// WHEN ValidateUpdate, ValidateUpdateV1Beta1 func is called
// THEN if we try to disable rancher backup an Error is thrown
func TestValidateUpdateMethods(t *testing.T) {
	err := NewComponent().ValidateUpdate(rancherBackupEnabledCR, &v1alpha1.Verrazzano{})
	assert.Error(t, err)

	v1beta1Vz := &v1beta1.Verrazzano{}
	_ = rancherBackupEnabledCR.ConvertTo(v1beta1Vz)
	err = NewComponent().ValidateUpdateV1Beta1(v1beta1Vz, &v1beta1.Verrazzano{})
	assert.Error(t, err)
}

// TestIsReady verifies component IsReady checks presence of the
// Rancher Backup operator deployment
func TestIsReady(t *testing.T) {
	fakeReadyClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentName,
				Namespace:  ComponentNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
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
					"app":               ComponentName,
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

	fakeUnReadyClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
		},
	).Build()

	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		isReady  bool
	}{
		// GIVEN a verrazzano CR with required deployments
		// WHEN IsReady func is called
		// THEN true is returned
		{
			"should be ready",
			spi.NewFakeContext(fakeReadyClient, &v1alpha1.Verrazzano{}, nil, true),
			true,
		},
		// GIVEN a verrazzano CR with no deployments
		// WHEN IsReady func is called
		// THEN false is returned
		{
			"should not be ready due to deployment",
			spi.NewFakeContext(fakeUnReadyClient, &v1alpha1.Verrazzano{}, nil, true),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.isReady, NewComponent().IsReady(tt.ctx))
		})
	}
}

// GIVEN a verrazzano CR
// WHEN PreInstall func is called it tries to install crds
// THEN true is returned if successful else false is returned in case of failure
func TestPreInstall(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithoutInstallation)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentName,
				Finalizers: []string{"fake-finalizer"},
			},
		},
	).Build()
	// Expect error as it will not be able to locate crds
	ctx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.Error(t, NewComponent().PreInstall(ctx))

	// Setting right path so that crds can be located and it returns no error
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..",
	})
	ctx = spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.NoError(t, NewComponent().PreInstall(ctx))
}

// GIVEN a verrazzano CR
// WHEN MonitorOverrides func is called it checks if InstallOverrides are set
// THEN true is returned if InstallOverrides are set else false
func TestMonitorOverrides(t *testing.T) {
	// Returns false if Backup component is not enabled
	ctx := spi.NewFakeContext(nil, &v1alpha1.Verrazzano{}, nil, false)
	assert.False(t, NewComponent().MonitorOverrides(ctx))

	// Returns true if Backup component is enabled
	ctx = spi.NewFakeContext(nil, rancherBackupEnabledCR, nil, false)
	assert.True(t, NewComponent().MonitorOverrides(ctx))

	// Check if Monitoring changes value is returned as set
	ctx = spi.NewFakeContext(nil, &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				RancherBackup: &v1alpha1.RancherBackupComponent{
					Enabled: &enabled,
					InstallOverrides: v1alpha1.InstallOverrides{
						MonitorChanges: &enabled,
						ValueOverrides: nil,
					},
				},
			},
		},
	}, nil, false)
	assert.True(t, NewComponent().MonitorOverrides(ctx))
}

func TestGetName(t *testing.T) {
	v := NewComponent()
	assert.Equal(t, ComponentName, v.Name())
	assert.Equal(t, ComponentJSONName, v.GetJSONName())
}

// TestPostUninstall tests the postUninstall function
// GIVEN a call to postUninstall
//
//	WHEN the cattle-resources-namespace  namespace exists with a finalizer
//	THEN true is returned and cattle-resources-namespace namespace is deleted
func TestPostUninstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
		},
	).Build()

	var iComp rancherBackupHelmComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.NoError(t, iComp.PostUninstall(compContext))

	// Validate that the namespace does not exist
	ns := corev1.Namespace{}
	err := compContext.Client().Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))
}

func TestValidateMethods(t *testing.T) {
	tests := []struct {
		name    string
		vz      *v1alpha1.Verrazzano
		wantErr bool
	}{
		{
			name:    "singleOverride",
			vz:      getSingleOverrideCR(),
			wantErr: false,
		},
		{
			name:    "multipleOverrides",
			vz:      getMultipleOverrideCR(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.ValidateUpdate(&v1alpha1.Verrazzano{}, tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			v1beta1Vz := &v1beta1.Verrazzano{}
			err := tt.vz.ConvertTo(v1beta1Vz)
			assert.NoError(t, err)
			if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.ValidateUpdateV1Beta1(&v1beta1.Verrazzano{}, v1beta1Vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getSingleOverrideCR() *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				JaegerOperator: &v1alpha1.JaegerOperatorComponent{
					Enabled: &enabled,
					InstallOverrides: v1alpha1.InstallOverrides{
						ValueOverrides: []v1alpha1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(validOverrideJSON),
								},
							},
						},
					},
				},
			},
		},
	}
}

func getMultipleOverrideCR() *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				RancherBackup: &v1alpha1.RancherBackupComponent{
					Enabled: &enabled,
					InstallOverrides: v1alpha1.InstallOverrides{
						ValueOverrides: []v1alpha1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(validOverrideJSON),
								},
								ConfigMapRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "overrideConfigMapSecretName",
									},
									Key: "Key",
								},
							},
						},
					},
				},
			},
		},
	}
}
