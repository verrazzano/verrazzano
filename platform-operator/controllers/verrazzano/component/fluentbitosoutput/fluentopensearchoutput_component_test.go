// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentbitosoutput

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
)

// TestReconcile tests Reconcile function for the FluentbitOpensearchOutput component
// GIVEN a FluentbitOpensearchOutput component
// WHEN I call Reconcile function with FluentbitOpensearchOutput context
// THEN nil is returned, if FluentbitOpensearchOutput is successfully reconciled; otherwise, error will be returned.
func TestReconcile(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	cr := getFluentOSOutputCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, true)
	fakeCtxWithErr := spi.NewFakeContext(fakeClient, cr, nil, false)
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
					Description:   "Named Release",
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
			"Successfully reconcile of fluentOpensearchOutput",
			args{
				fakeContext,
			},
			false,
		},
		{
			"error during reconcile of fluentOpensearchOutput",
			args{
				fakeCtxWithErr,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.Reconcile(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPreUpgradeAndInstall tests PreUpgrade and  PreInstall function for the FluentbitOpensearchOutput component
// GIVEN a FluentbitOpensearchOutput component
// WHEN I call Reconcile function with FluentbitOpensearchOutput context
// THEN nil is returned, if all PreInstall And PreUpgrade tasks of FluentbitOpensearchOutput are successfully executed; otherwise, error will be returned.
func TestPreUpgradeAndPreInstall(t *testing.T) {
	type args struct {
		ctx spi.ComponentContext
	}
	cr := getFluentOSOutputCR(true)
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	faktClientWithSecret := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{
			Type:       "helm.sh/release.v1",
			ObjectMeta: v1.ObjectMeta{Name: globalconst.VerrazzanoESInternal, Namespace: globalconst.VerrazzanoSystemNamespace},
		},
	).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	fakeCtxWithSecret := spi.NewFakeContext(faktClientWithSecret, cr, nil, true)
	config.SetDefaultBomFilePath(testBomFilePath)
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
					Description:   "FluentOpensearchOutput release",
				},
				Version: 1,
			}
		})
	})
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})

	defer helmcli.SetDefaultActionConfigFunction()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"PreUpgrade/PreInstall with no secret",
			args{
				fakeContext,
			},
			true,
		},
		{
			"PreUpgrade/PreInstall with OS secret",
			args{
				fakeCtxWithSecret,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.PreUpgrade(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("PreUpgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.PreInstall(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("PreUpgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetOverrides tests getOverrides functions for the FluentbitOpensearchOutput component
// GIVEN a FluentbitOpensearchOutput component
// WHEN I call GetOverrides function with Verrazzano CR object
// THEN all installed overrides available in Verrazzano CR for the FluentbitOpensearchOutput are returned.
func TestGetOverrides(t *testing.T) {
	type args struct {
		object runtime.Object
	}
	ref := &corev1.ConfigMapKeySelector{
		Key: "testCM",
	}
	oV1Beta1 := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := v1alpha1.InstallOverrides{
		ValueOverrides: []v1alpha1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			"TestGetOverrides with v1alpha1",
			args{
				&v1alpha1.Verrazzano{
					Spec: v1alpha1.VerrazzanoSpec{
						Components: v1alpha1.ComponentSpec{
							FluentbitOpensearchOutput: &v1alpha1.FluentbitOpensearchOutputComponent{
								InstallOverrides: oV1Alpha1,
							},
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"TestGetOverrides with v1beta1",
			args{
				&v1beta1.Verrazzano{
					Spec: v1beta1.VerrazzanoSpec{
						Components: v1beta1.ComponentSpec{
							FluentbitOpensearchOutput: &v1beta1.FluentbitOpensearchOutputComponent{
								InstallOverrides: oV1Beta1,
							},
						},
					},
				},
			},
			oV1Beta1.ValueOverrides,
		},
		{
			"no overrides when component is nil",
			args{&v1beta1.Verrazzano{}},
			[]v1beta1.Overrides{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOverrides(tt.args.object); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMonitorOverrides tests the MonitorOverrides function for the FluentbitOpensearchOutput component
// GIVEN a FluentbitOpensearchOutput component
// WHEN I call MonitorOverrides with Verrazzano CR
// THEN true is returned if MonitorChanges is enabled in Verrazzano CR; otherwise, false will be returned.
func TestMonitorOverrides(t *testing.T) {
	component := NewComponent()
	cr := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentbitOpensearchOutput: &v1alpha1.FluentbitOpensearchOutputComponent{
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
				FluentbitOpensearchOutput: &v1alpha1.FluentbitOpensearchOutputComponent{
					InstallOverrides: v1alpha1.InstallOverrides{
						MonitorChanges: common.BoolPtr(false),
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeContext := spi.NewFakeContext(fakeClient, cr, nil, false)
	fakeCtxWithNoOverride := spi.NewFakeContext(fakeClient, disableMonitorOverridesCR, nil, false)
	monitor := component.MonitorOverrides(fakeContext)
	assert.True(t, monitor, "Monitoring of install overrides should be enabled")
	monitor = component.MonitorOverrides(fakeCtxWithNoOverride)
	assert.False(t, monitor, "Monitoring of install overrides should be disabled")
}

// TestAppendOverrides test the AppendOverrides function for fluentbitOpensearchOutput.
// GIVEN a FluentbitOpensearchOutput component
// WHEN I call AppendOverrides function with FluentbitOpensearchOutput context
// THEN Override should be from the Cluster registration secret, otherwise no overrides should be there.
func TestAppendOverrides(t *testing.T) {
	const hostName = "xyz.com"
	const port = "443"
	const testURL = "https://" + hostName + ":" + port
	registrationSecret := createTestRegistrationSecret(map[string]string{
		constants.OpensearchURLData: testURL,
	})
	expectedKVSWithOverride := []bom.KeyValue{
		{Key: OverrideApplicationHostKey, Value: hostName},
		{Key: OverrideSystemHostKey, Value: hostName},
		{Key: OverrideApplicationPortKey, Value: port},
		{Key: OverrideSystemPortKey, Value: port},
		{Key: OverrideApplicationPasswordKey, Value: constants.MCRegistrationSecret},
		{Key: OverrideSystemPasswordKey, Value: constants.MCRegistrationSecret},
		{Key: OverrideApplicationUserKey, Value: constants.MCRegistrationSecret},
		{Key: OverrideSystemUserKey, Value: constants.MCRegistrationSecret},
		{Key: OverrideSystemTLSKey, Value: "true"},
		{Key: OverrideSystemCAFileKey, Value: CACertPath + "/" + CACertName},
		{Key: OverrideSystemCertKey, Value: FluentBitCertPath},
		{Key: OverrideApplicationTLSKey, Value: "true"},
		{Key: OverrideApplicationCAFileKey, Value: CACertPath + "/" + CACertName},
		{Key: OverrideApplicationCertKey, Value: FluentBitCertPath},
	}
	expectedKVS := []bom.KeyValue{}
	cr := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentbitOpensearchOutput: &v1alpha1.FluentbitOpensearchOutputComponent{},
			},
		},
	}
	var tests = []struct {
		name     string
		ctx      spi.ComponentContext
		expected []bom.KeyValue
	}{
		{
			"uses fluentbitOpensearchOutput URL and credentials if no registration secret",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), cr, nil, false),
			expectedKVS,
		},
		{
			"uses registration secret for overrides if present",
			spi.NewFakeContext(fake.NewClientBuilder().WithObjects(registrationSecret).Build(), cr, nil, false),
			expectedKVSWithOverride,
		},
	}

	for _, tt := range tests {
		got, err := AppendOverrides(tt.ctx, "", "", "", []bom.KeyValue{})
		if err != nil {
			t.Errorf("AppendOverrides() error = %v", err)
			return
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("AppendOverrides() got = %v, want %v", got, tt.expected)
		}
	}
}

// getFluentOSOutputCR returns the Verrazzano CR with enabled/disabled FluentbitOpensearchOutput based on the passed argument.
func getFluentOSOutputCR(enabled bool) *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				FluentbitOpensearchOutput: &v1alpha1.FluentbitOpensearchOutputComponent{
					Enabled: common.BoolPtr(enabled),
				},
			},
		},
	}
}

// createTestRegistrationSecret returns a registration secret for unit test purpose.
func createTestRegistrationSecret(kvs map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: ComponentNamespace,
		},
	}
	s.Data = map[string][]byte{}
	for k, v := range kvs {
		s.Data[k] = []byte(v)
	}
	return s
}
