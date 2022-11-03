// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestIsEnabled tests the AuthProxy IsEnabled call
// GIVEN a AuthProxy component
//
//	WHEN I call IsEnabled when all requirements are met
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
			name: "Test IsEnabled when using AuthProxy component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
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

// TestGetIngressNames tests the AuthProxy GetIngressNames call
// GIVEN a AuthProxy component
//
//	WHEN I call GetIngressNames
//	THEN the correct list of names is returned
func TestGetIngressNames(t *testing.T) {
	ingressNames := NewComponent().GetIngressNames(nil)
	assert.True(t, len(ingressNames) == 1)
	assert.Equal(t, constants.VzConsoleIngress, ingressNames[0].Name)
	assert.Equal(t, ComponentNamespace, ingressNames[0].Namespace)
}

// TestValidateUpdate tests the AuthProxy ValidateUpdate call for v1alpha1.Verrazzano
func TestValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with auth proxy component disabled,
		// WHEN I call update the VZ CR to enable auth proxy component
		// THEN the update succeeds with no errors.
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		// GIVEN a VZ CR with auth proxy component enabled,
		// WHEN I call update the VZ CR to disable auth proxy component
		// THEN the update fails with an error.
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with auth proxy component,
		// WHEN I call update with no change to the auth proxy component
		// THEN the update succeeds and no error is returned.
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
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

// TestValidateUpdateV1beta1 tests the AuthProxy ValidateUpdate call for v1beta1.Verrazzano
func TestValidateUpdateV1beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with auth proxy component disabled,
		// WHEN I call update the VZ CR to enable auth proxy component
		// THEN the update succeeds with no error.
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						AuthProxy: &v1beta1.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		// GIVEN a VZ CR with auth proxy component enabled,
		// WHEN I call update the VZ CR to disable auth proxy component
		// THEN the update fails with an error.
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						AuthProxy: &v1beta1.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with auth proxy component,
		// WHEN I call update with no change to the auth proxy component
		// THEN the update succeeds and no error is returned.
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

// TestUninstallHelmChartInstalled tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//
//	WHEN I call Uninstall with the Fluentd helm chart installed
//	THEN no error is returned
func TestUninstallHelmChartInstalled(t *testing.T) {
	helmcli.SetCmdRunner(os.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte{},
		Err:    nil,
	})
	defer helmcli.SetDefaultRunner()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestUninstallHelmChartNotInstalled tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//
//	WHEN I call Uninstall with the Fluentd helm chart not installed
//	THEN no error is returned
func TestUninstallHelmChartNotInstalled(t *testing.T) {
	helmcli.SetCmdRunner(os.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte{},
		Err:    fmt.Errorf("Not installed"),
	})
	defer helmcli.SetDefaultRunner()
	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestIsReady tests the IsReady is available and ready
//
//	GIVEN a AuthProxy component
//	WHEN IsAvailable is called
//	THEN True is returned if AuthProxy is ready  and
//	    False is returned if AuthProxy  is ready.
func TestIsReady(t *testing.T) {
	objectMeta := metav1.ObjectMeta{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	readyAndAvailableClient := fake.NewClientBuilder().
		WithObjects(&appsv1.Deployment{
			ObjectMeta: objectMeta,
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				ReadyReplicas:     1,
				UpdatedReplicas:   1,
				AvailableReplicas: 1,
			},
		},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ComponentNamespace,
					Name:      ComponentName + "-19e3je32-m6mbr",
					Labels: map[string]string{
						"pod-template-hash": "19e3je32",
						"app":               ComponentName,
					},
				},
			},
			&appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   ComponentNamespace,
					Name:        ComponentName + "-19e3je32",
					Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
				},
			},
		).Build()
	notAvailableClient := fake.NewClientBuilder().
		WithObjects(&appsv1.Deployment{
			ObjectMeta: objectMeta,
			Status: appsv1.DeploymentStatus{
				Replicas:        1,
				ReadyReplicas:   0,
				UpdatedReplicas: 0,
			},
		}).Build()
	tests := []struct {
		name       string
		client     clipkg.Client
		actualCR   vzapi.Verrazzano
		expectTrue bool
		reason     string
	}{
		{
			name:       "Test IsReady when AuthProxy component pod is ready",
			client:     readyAndAvailableClient,
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			name:       "Test IsReady when AuthProxy component pod is not ready",
			client:     notAvailableClient,
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &tests[i].actualCR, nil, true, profilesRelativePath)
			isAvailable := NewComponent().IsReady(ctx)
			if tt.expectTrue {
				assert.True(t, isAvailable)
			} else {
				assert.False(t, isAvailable)
			}
		})
	}
}

// TestIsReadyHelmError tests IsReady
//
//	GIVEN a AuthProxy component
//	WHEN IsReady is called
//	THEN False is returned if helm CLI throws error
func TestIsReadyHelmError(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	tests := []struct {
		name       string
		client     clipkg.Client
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Test IsReady when HelmComponent throw errors",
			client:     client,
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &tests[i].actualCR, nil, false, profilesRelativePath)
			isAvailable := NewComponent().IsReady(ctx)
			if tt.expectTrue {
				assert.True(t, isAvailable)
			} else {
				assert.False(t, isAvailable)
			}
		})
	}
}

// TestMonitorOverrides test the MonitorOverrides to confirm monitoring of install overrides is enabled or not
//
//	GIVEN a default VZ CR with auth proxy component
//	WHEN  MonitorOverrides is called
//	THEN  returns True if monitoring of install overrides is enabled and False otherwise
func TestMonitorOverrides(t *testing.T) {
	disabled := false
	enabled := true
	tests := []struct {
		name     string
		actualCR *vzapi.Verrazzano
		want     bool
	}{
		{
			name: "Test MonitorOverrides when Authproxy is disabled in the spec",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Test MonitorOverrides when Authproxy is nil",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "Test MonitorOverrides when MonitorOverrides is enabled in the spec",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled:          &enabled,
							InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &enabled},
						},
					},
				},
			},
			want: true,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			ctx := spi.NewFakeContext(nil, tests[i].actualCR, nil, true)
			if got := c.MonitorOverrides(ctx); got != tt.want {
				t.Errorf("MonitorOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPreInstall test the PreInstall to check all the
// pre-install operations are successful executed or not.
//
//	GIVEN an Authproxy component
//	WHEN I call PreInstall with defaults
//	THEN no error is returned
func TestPreInstall(t *testing.T) {
	defer helmcli.SetDefaultRunner()
	helmCliNoError := func() {
		helmcli.SetCmdRunner(os.GenericTestRunner{
			StdOut: []byte(""),
			StdErr: []byte{},
			Err:    nil,
		})
	}
	helmCliError := func() {
		helmcli.SetCmdRunner(os.GenericTestRunner{
			StdOut: []byte(""),
			StdErr: []byte{},
			Err:    fmt.Errorf("not found"),
		})
	}
	tests := []struct {
		name        string
		client      clipkg.Client
		helmcliFunc func()
		actualCR    vzapi.Verrazzano
		expectTrue  bool
	}{
		{
			name:        "Test PreInstall when AuthProxy is already installed and no error",
			client:      fake.NewClientBuilder().Build(),
			actualCR:    vzapi.Verrazzano{},
			expectTrue:  false,
			helmcliFunc: helmCliNoError,
		},
		{
			name:        "Test PreInstall when AuthProxy is not installed and no error",
			client:      fake.NewClientBuilder().Build(),
			actualCR:    vzapi.Verrazzano{},
			expectTrue:  false,
			helmcliFunc: helmCliError,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.helmcliFunc()
			ctx := spi.NewFakeContext(tt.client, &tests[i].actualCR, nil, false, profilesRelativePath)
			err := NewComponent().PreInstall(ctx)
			if tt.expectTrue {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPreUpgrade test the PreUpgrade to check all the pre upgrade operations
// are successful executed or not.
//
//	GIVEN an Authproxy component
//	WHEN I call PreUpgrade with defaults
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	tests := []struct {
		name       string
		client     clipkg.Client
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Test PreUpgrade when there is no error",
			client:     fake.NewClientBuilder().Build(),
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &tests[i].actualCR, nil, false, profilesRelativePath)
			err := NewComponent().PreUpgrade(ctx)
			if tt.expectTrue {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
