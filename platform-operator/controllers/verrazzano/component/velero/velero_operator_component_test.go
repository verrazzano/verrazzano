// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profilesRelativePath = "../../../../manifests/profiles"
	validOverrideJSON    = "{\"serviceAccount\": {\"create\": false}}"
)

var enabled = true
var veleroEnabledCR = &v1alpha1.Verrazzano{
	Spec: v1alpha1.VerrazzanoSpec{
		Components: v1alpha1.ComponentSpec{
			Velero: &v1alpha1.VeleroComponent{
				Enabled: &enabled,
			},
		},
	},
}

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestIsEnabled tests the IsEnabled function for the Velero Operator component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   v1alpha1.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Velero Operator component
			// THEN the call returns false
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   v1alpha1.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Velero Operator enabled
			// WHEN we call IsReady on the Velero Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when Velero Operator component set to enabled",
			actualCR:   *veleroEnabledCR,
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Velero Operator disabled
			// WHEN we call IsReady on the Velero Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Velero Operator component set to disabled",
			actualCR: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Velero: &v1alpha1.VeleroComponent{
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
// Velero operator deployment
func TestIsInstalled(t *testing.T) {
	var tests = []struct {
		name        string
		client      crtclient.Client
		isInstalled bool
	}{
		{
			"installed when Velero deployment is present",
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
			"not installed when Velero deployment is absent",
			fake.NewClientBuilder().WithScheme(testScheme).Build(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, veleroEnabledCR, nil, false)
			installed, err := NewComponent().IsInstalled(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.isInstalled, installed)
		})
	}
}

func TestInstallUpgrade(t *testing.T) {
	defer config.Set(config.Get())
	v := NewComponent()
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})

	helm.SetCmdRunner(genericTestRunner{
		stdOut: []byte(""),
		stdErr: []byte{},
		err:    nil,
	})
	defer helm.SetDefaultRunner()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(veleroEnabledCR).Build()
	ctx := spi.NewFakeContext(client, veleroEnabledCR, nil, false)
	err := v.Install(ctx)
	assert.NoError(t, err)
	err = v.Upgrade(ctx)
	assert.NoError(t, err)
	err = v.Reconcile(ctx)
	assert.NoError(t, err)
}

func TestGetName(t *testing.T) {
	v := NewComponent()
	assert.Equal(t, ComponentName, v.Name())
	assert.Equal(t, ComponentJSONName, v.GetJSONName())
}

// TestPostUninstall tests the PostUninstall function
// GIVEN a call to PostUninstall
//
//	WHEN the velero namespace exists with a finalizer
//	THEN true is returned and velero namespace is deleted
func TestPostUninstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       ComponentNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
		},
	).Build()

	var iComp veleroHelmComponent
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.NoError(t, iComp.PostUninstall(compContext))

	// Validate that the namespace does not exist
	ns := corev1.Namespace{}
	err := compContext.Client().Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))
}

// TestValidateMethods tests ValidateInstall, ValidateUpdate, ValidateInstallV1Beta1 and  ValidateUpdateV1Beta1
//
//		GIVEN VZ CR with install single and multi overrides
//
//	 WHEN ValidateInstall, ValidateUpdate, ValidateInstallV1Beta1 and  ValidateUpdateV1Beta1 are called
//	 THEN  if install overrides are not invalid, error is returned else no error is returned
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

// TestValidateUpdate tests the ValidateUpdate and ValidateUpdateV1Beta1
func TestValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		vz      *v1alpha1.Verrazzano
		wantErr bool
	}{
		//   GIVEN VZ CR with install single overrides
		//	 WHEN ValidateUpdate and ValidateUpdateV1Beta1 are called
		//	 THEN  if Velero is disabled in the new CR, ValidateUpdate and ValidateUpdateV1Beta1
		//         will throw error
		{
			name:    "singleOverride",
			vz:      getSingleOverrideCR(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			// test ValidateUpdate when Velero is disabled in the new CR
			if err := c.ValidateUpdate(tt.vz, &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Velero: &v1alpha1.VeleroComponent{
							Enabled: &disabled,
						},
					},
				},
			}); err == nil {
				t.Errorf("TestValidateUpdate: ValidateUpdate() error = %v, wantErr %v", err, true)
			}
			v1beta1Vz := &v1beta1.Verrazzano{}
			err := tt.vz.ConvertTo(v1beta1Vz)
			assert.NoError(t, err)
			// test ValidateUpdateV1Beta1 when Velero is disabled in the new CR
			if err = c.ValidateUpdateV1Beta1(v1beta1Vz, &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Velero: &v1beta1.VeleroComponent{
							Enabled: &disabled,
						},
					},
				},
			}); err == nil {
				t.Errorf("TestValidateUpdate: ValidateUpdateV1Beta1() error = %v, wantErr %v", err, true)
			}
		})
	}
}

// TestPreInstall tests the PreInstall function for the Velero Operator
func TestPreInstall(t *testing.T) {
	enable := true
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	// Expect a failed call to fetch the Velero namespace
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: ComponentNamespace}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("internal server error"))
	tests := []struct {
		name    string
		ctx     spi.ComponentContext
		wantErr bool
	}{
		// GIVEN default VZ CR
		// WHEN PreInstall is called
		// THEN Velero component namespace is created and no error is thrown
		{
			"PreInstallWithNoError",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{}, nil, false),
			false,
		},
		// GIVEN default VZ CR with istio sidecar injection enabled
		// WHEN PreInstall is called
		// THEN Velero component namespace is created and no error is thrown
		{
			"PreInstallIstioInjectionEnabledWithNoError",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Enabled:          &enable,
							InjectionEnabled: &enable,
						},
					},
				},
			}, nil, false),
			false,
		},
		// GIVEN default VZ CR
		// WHEN PreInstall is called
		// THEN error is thrown if there is any error from the client
		{
			"PreInstallWithError",
			spi.NewFakeContext(mockClient, &v1alpha1.Verrazzano{}, nil, false),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewComponent().PreInstall(tt.ctx); (err != nil) != tt.wantErr {
				t.Errorf("PreInstall error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMonitorOverrides test the MonitorOverrides to confirm monitoring of install overrides is enabled or not
//
//	GIVEN a default VZ CR with Velero component
//	WHEN  MonitorOverrides is called
//	THEN  returns True if monitoring of install overrides is enabled and False otherwise
func TestMonitorOverrides(t *testing.T) {
	disabled := false
	tests := []struct {
		name     string
		actualCR *v1alpha1.Verrazzano
		want     bool
	}{
		{
			name: "Test MonitorOverrides when Velero is disabled in the spec",
			actualCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Velero: &v1alpha1.VeleroComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Test MonitorOverrides when Velero component is nil",
			actualCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Velero: nil,
					},
				},
			},
			want: false,
		},
		{
			name: "Test MonitorOverrides when MonitorOverrides is enabled in the spec",
			actualCR: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Velero: &v1alpha1.VeleroComponent{
							Enabled:          &enabled,
							InstallOverrides: v1alpha1.InstallOverrides{MonitorChanges: &enabled},
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

func getSingleOverrideCR() *v1alpha1.Verrazzano {
	return &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Velero: &v1alpha1.VeleroComponent{
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
				Velero: &v1alpha1.VeleroComponent{
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
