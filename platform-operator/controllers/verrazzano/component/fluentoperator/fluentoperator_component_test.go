// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentoperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

func TestValidateInstall(t *testing.T) {
	component := NewComponent()
	vz := &v1alpha1.Verrazzano{}
	err := component.ValidateInstall(vz)
	assert.NoError(t, err, "ValidateInstall should not return an error")

}

func TestIsEnabled(t *testing.T) {
	component := NewComponent()
	cr := getFluentOperatorCR(true)
	enabled := component.IsEnabled(cr)
	assert.True(t, enabled, "FluentOperator component should be enabled")
}

func TestMonitorOverrides(t *testing.T) {
	component := NewComponent()
	cr := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentOperator: &v1alpha1.FluentOperatorComponent{
					InstallOverrides: v1alpha1.InstallOverrides{
						MonitorChanges: common.BoolPtr(true),
					},
				},
			},
		},
	}
	disableMonitorOverridesCR := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentOperator: &v1alpha1.FluentOperatorComponent{
					InstallOverrides: v1alpha1.InstallOverrides{
						MonitorChanges: common.BoolPtr(false),
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	fakeCtxWithNoOverride := spi.NewFakeContext(fakeClient, disableMonitorOverridesCR, nil, false)
	monitor := component.MonitorOverrides(fakeContext)
	assert.True(t, monitor, "Monitoring of install overrides should be enabled")
	monitor = component.MonitorOverrides(fakeCtxWithNoOverride)
	assert.Falsef(t, monitor, "Monitoring of install overrides should be disabled")
}

func TestValidateUpdate(t *testing.T) {
	type args struct {
		old *v1alpha1.Verrazzano
		new *v1alpha1.Verrazzano
	}
	enabledFluentOperator := getFluentOperatorCR(true)
	disabledFluentOperator := getFluentOperatorCR(false)
	invalidOverrideCR := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentOperator: &v1alpha1.FluentOperatorComponent{
					Enabled: common.BoolPtr(true),
					InstallOverrides: v1alpha1.InstallOverrides{
						ValueOverrides: []v1alpha1.Overrides{
							{
								ConfigMapRef: &corev1.ConfigMapKeySelector{},
								SecretRef:    &corev1.SecretKeySelector{},
							},
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Test enabling Fluent Operator",
			args{
				disabledFluentOperator,
				enabledFluentOperator,
			},
			false,
		},
		{
			"Test enabling Fluent Operator",
			args{
				enabledFluentOperator,
				disabledFluentOperator,
			},
			true,
		},
		{
			"Test invalid Overrides",
			args{
				enabledFluentOperator,
				invalidOverrideCR,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			err := c.ValidateUpdate(tt.args.old, tt.args.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func getFluentOperatorCR(enabled bool) *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentOperator: &v1alpha1.FluentOperatorComponent{
					Enabled: common.BoolPtr(enabled),
				},
			},
		},
	}
}
func TestInstallAndUpgrade(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	defer func() {
		config.TestThirdPartyManifestDir = ""
	}()
	cr := getFluentOperatorCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})

	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helmcli.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
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
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Successful Install/Upgrade of FluentOperator",
			args{
				fakeContext,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			err := c.PostInstall(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostInstall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = c.PostUpgrade(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostUpgrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			config.TestThirdPartyManifestDir = "../../../../thirdparty/manifests"
			err = c.PreInstall(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PreInstall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = c.Install(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = c.PreUpgrade(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PreUpgrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = c.Upgrade(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upgrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	cr := getFluentOperatorCR(true)
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helmcli.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"k8s-app": ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, true)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Successful reconcile",
			args{fakeContext},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			err := c.Reconcile(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestIsReady(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	cr := getFluentOperatorCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"When Fluent Operator is ready",
			args{
				fakeContext,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			assert.Equalf(t, tt.want, c.IsReady(tt.args.ctx), "IsReady(%v)", tt.args.ctx)
		})
	}
}

func TestUninstall(t *testing.T) {
	type args struct {
		context spi.ComponentContext
	}
	cr := getFluentOperatorCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, true)
	contextWithErr := spi.NewFakeContext(fakeClient, cr, nil, false)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"UnInstall Fluent Operator",
			args{
				fakeContext,
			},
			false,
		},
		{
			"Error during unInstalling Fluent Operator",
			args{
				contextWithErr,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			defer helmcli.SetDefaultActionConfigFunction()
			helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
				return helmcli.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
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
			err := c.Uninstall(tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("Uninstall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestIsInstalled(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	cr := getFluentOperatorCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"k8s-app": ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	clientWithNoFluentOperator := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	ctxWithNoFluentOperator := spi.NewFakeContext(clientWithNoFluentOperator, cr, nil, false)
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"Fluent Operator is Installed",
			args{
				fakeContext,
			},
			true,
			false,
		},
		{
			"Fluent Operator is not Installed",
			args{
				ctxWithNoFluentOperator,
			},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			got, err := c.IsInstalled(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equalf(t, tt.want, got, "IsInstalled(%v)", tt.args.ctx)
		})
	}
}
