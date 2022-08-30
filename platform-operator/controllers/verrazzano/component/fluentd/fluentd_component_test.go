// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme = runtime.NewScheme()
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
)

var enabled = true
var notEnabled = false
var fluentdEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Fluentd: &vzapi.FluentdComponent{
				Enabled: &enabled,
			},
		},
	},
}

var vzEsInternalSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      globalconst.VerrazzanoESInternal,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
}

func init() {
	_ = vzapi.AddToScheme(testScheme)
	_ = clientgoscheme.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

func TestValidateUpdate(t *testing.T) {
	disabled := false
	sec := getFakeSecret("TestValidateUpdate-es-sec")
	defer func() { getControllerRuntimeClient = getClient }()
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable-fluentd",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{Enabled: &disabled},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-fluentd-oci",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							OCI: &vzapi.OciLoggingConfiguration{
								APISecret: "secret",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-fluentd-es-secret",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							ElasticsearchSecret: sec.Name,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-fluentd-es-url",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							ElasticsearchURL: "url",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-fluentd-extravolume",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							ExtraVolumeMounts: []vzapi.VolumeMount{{Source: "foo"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid-fluentd-extravolume",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							ExtraVolumeMounts: []vzapi.VolumeMount{{Source: "/root/.oci"}},
						},
					},
				},
			},
			wantErr: true,
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

func TestValidateInstall(t *testing.T) {
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "FluentdComponent empty",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "FluentdComponent empty",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "oci and ext-es",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							OCI:              &vzapi.OciLoggingConfiguration{},
							ElasticsearchURL: "https://url",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostUpgrade tests the Fluentd PostUpgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN a Verrazzano component upgrading from 1.1.0 to 1.4.0
//  WHEN I call PostUpgrade
//  THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := getFakeComponentContext(c)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}

// TestPreInstall tests the Fluentd PreInstall call
// GIVEN a Fluentd component
//  WHEN I call PreInstall when dependencies are met
//  THEN no error is returned. Otherwise, return error.
func TestPreInstall(t *testing.T) {
	var tests = []struct {
		name   string
		spec   *vzapi.Verrazzano
		client client.Client
		err    error
	}{
		{
			"should fail when verrazzano-es-internal secret does not exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(),
			ctrlerrors.RetryableError{Source: ComponentName},
		},
		{
			"should pass when verrazzano-es-internal secret does exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(vzEsInternalSecret),
			nil,
		},
		{
			"always nil error when keycloak is disabled",
			keycloakDisabledCR,
			createFakeClient(),
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, nil, false)
			err := NewComponent().PreInstall(ctx)
			if tt.err != nil {
				assert.Error(t, err)
				assert.IsTypef(t, tt.err, err, "")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetOverrides(t *testing.T) {
	ref := &corev1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := vzapi.InstallOverrides{
		ValueOverrides: []vzapi.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	var tests = []struct {
		name string
		cr   runtime.Object
		res  interface{}
	}{
		{
			"overrides when component not nil, v1alpha1",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Fluentd: &vzapi.FluentdComponent{
							InstallOverrides: oV1Alpha1,
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"Empty overrides when component nil",
			&v1beta1.Verrazzano{},
			[]v1beta1.Overrides{},
		},
		{
			"overrides when component not nil",
			&v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Fluentd: &v1beta1.FluentdComponent{
							InstallOverrides: o,
						},
					},
				},
			},
			o.ValueOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}

func createFakeClient(extraObjs ...client.Object) client.Client {
	objs := []client.Object{}
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build()
	return c
}

// TestInstall tests the Verrazzano Install call
// GIVEN a Verrazzano component
//  WHEN I call Install when dependencies are met
//  THEN no error is returned
func TestInstall(t *testing.T) {
	c := createFakeClient()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{ElasticsearchSecret: vzapi.OciConfigSecretFile},
			},
		},
	}, nil, false)
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

// fakeUpgrade override the upgrade function during unit tests
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

// TestPreUpgrade tests the Verrazzano PreUpgrade call
// GIVEN a Verrazzano component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned. Otherwise, return error.
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../helm_config"

	var tests = []struct {
		name   string
		spec   *vzapi.Verrazzano
		client client.Client
		err    error
	}{
		{
			"should fail when verrazzano-es-internal secret does not exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(),
			ctrlerrors.RetryableError{Source: ComponentName},
		},
		{
			"should pass when verrazzano-es-internal secret does exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(vzEsInternalSecret),
			nil,
		},
		{
			"always nil error when keycloak is disabled",
			keycloakDisabledCR,
			createFakeClient(),
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, nil, false)
			err := NewComponent().PreUpgrade(ctx)
			if tt.err != nil {
				assert.Error(t, err)
				assert.IsTypef(t, tt.err, err, "")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpgrade tests the Fluentd Upgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN a Fluentd component upgrading from 1.1.0 to 1.4.0
//  WHEN I call Upgrade
//  THEN no error is returned
func TestUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := getFakeComponentContext(c)
	config.SetDefaultBomFilePath(testBomFilePath)
	helmcli.SetCmdRunner(os.GenericTestRunner{})
	defer helmcli.SetDefaultRunner()
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	helmcli.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helmcli.ChartStatusDeployed, nil
	})
	defer helmcli.SetDefaultChartStateFunction()
	err := NewComponent().Upgrade(ctx)
	assert.NoError(t, err)
}

// TestIsInstalled tests the IsInstalled function
// GIVEN a call to IsInstalled
//  WHEN the daemonset object is found
//  THEN true is returned. Otherwise, return false.
func TestIsInstalled(t *testing.T) {
	var tests = []struct {
		name        string
		client      client.Client
		isInstalled bool
	}{
		{
			"installed when jaeger deployment is present",
			fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			).Build(),
			true,
		},
		{
			"not installed when jaeger deployment is absent",
			fake.NewClientBuilder().WithScheme(testScheme).Build(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, fluentdEnabledCR, nil, false)
			installed, err := NewComponent().IsInstalled(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.isInstalled, installed)
		})
	}
}

// TestUninstallHelmChartInstalled tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//  WHEN I call Uninstall with the Fluentd helm chart installed
//  THEN no error is returned
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
//  WHEN I call Uninstall with the Fluentd helm chart not installed
//  THEN no error is returned
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

// TestUninstallResources tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//  WHEN I call Uninstall with the Fluentd helm chart not installed
//  THEN ensure that all Fluentd resources are explicity deleted
func TestUninstallResources(t *testing.T) {
	helmcli.SetCmdRunner(os.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte{},
		Err:    fmt.Errorf("Not installed"),
	})
	defer helmcli.SetDefaultRunner()

	clusterRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: ComponentName}}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ComponentName}}
	configMap1 := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "fluentd-config"}}
	configMap2 := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "fluentd-es-config"}}
	configMap3 := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "fluentd-init"}}
	daemonset := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: ComponentName}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: ComponentName}}
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: ComponentName}}

	c := fake.NewClientBuilder().WithScheme(clientgoscheme.Scheme).WithObjects(
		clusterRole,
		clusterRoleBinding,
		configMap1,
		configMap2,
		configMap3,
		daemonset,
		service,
		serviceAccount,
	).Build()

	err := NewComponent().Uninstall(spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)

	// Assert that the resources have been deleted
	err = c.Get(context.TODO(), types.NamespacedName{Name: ComponentName}, clusterRole)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Name: ComponentName}, clusterRoleBinding)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, configMap1)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, configMap2)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, configMap3)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, daemonset)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, service)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, serviceAccount)
	assert.True(t, errors.IsNotFound(err))
}

func getFakeComponentContext(c client.WithWatch) spi.ComponentContext {
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version: "v1.4.0",
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{ElasticsearchSecret: vzapi.OciConfigSecretFile},
			},
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, nil, false)
	return ctx
}
