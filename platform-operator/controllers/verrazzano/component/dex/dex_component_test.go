// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"context"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDexReady tests IsReady for the Dex component
// GIVEN a call to IsReady
// WHEN the VZ CR is populated
// THEN a boolean is returned
func TestDexReady(t *testing.T) {
	trueValue := true
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
					Name:      ComponentName + "687b47899b-fbt9p",
					Labels: map[string]string{
						"pod-template-hash": "687b47899b",
						"app":               ComponentName,
					},
				},
			},
			&appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   ComponentNamespace,
					Name:        ComponentName + "-687b47899b",
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
			name:   "Dex component with expected replicas",
			client: readyAndAvailableClient,
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			name:   "Dex component without expected replicas",
			client: notAvailableClient,
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace(constants.DexIngress)
			defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

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

// TestDexEnabled tests if Dex is enabled
// GIVEN a call to IsEnabled
// WHEN the VZ CR is populated
// THEN a boolean is returned
func TestDexEnabled(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Dex disabled in Verrazzano CR by default",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			name: "Dex disabled explicitly in Verrazzano CR",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
		{
			name: "Dex enabled in Verrazzano CR",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
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

// TestPreInstall tests the pre-install stage for the Dex component
// GIVEN a call to PreInstall
// WHEN Dex is enabled
// THEN the Dex namespace is created
func TestPreInstall(t *testing.T) {
	trueValue := true
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestNginxService()).Build()

	actualCR := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Dex: &vzapi.DexComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, &actualCR, nil, false, profilesRelativePath)
	assert.NoError(t, NewComponent().PreInstall(ctx))

	ns := corev1.Namespace{}
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: constants.DexNamespace}, &ns))
}

// TestPreUpgrade tests the pre-upgrade stage for the Dex component
// GIVEN a call to PreUpgrade
// WHEN Dex is enabled
// THEN the Dex namespace is created
func TestPreUpgrade(t *testing.T) {
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(testActionConfigWithInstalledDex)
	trueValue := true
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.Verrazzano,
			Namespace: constants.VerrazzanoSystemNamespace}}, createTestNginxService(),
	).Build()

	actualCR := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Dex: &vzapi.DexComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, &actualCR, nil, false, profilesRelativePath)
	err := NewComponent().PreUpgrade(ctx)
	assert.NoError(t, err)

	ns := corev1.Namespace{}
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: constants.DexNamespace}, &ns))
}

// TestValidateUpdateV1beta1 tests the Dex ValidateUpdate call for v1beta1.Verrazzano
func TestValidateUpdateV1beta1(t *testing.T) {
	disabled := false
	enabled := true
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with Dex component disabled,
		// WHEN I call update the VZ CR to enable Dex component
		// THEN the update succeeds with no error.
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Dex: &v1beta1.DexComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Dex: &v1beta1.DexComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			wantErr: false,
		},
		// GIVEN a VZ CR with Dex component enabled,
		// WHEN I call update the VZ CR to disable Dex component
		// THEN the update fails with an error.
		{
			name: "disable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Dex: &v1beta1.DexComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Dex: &v1beta1.DexComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with Dex component disabled,
		// WHEN I call update with no change to the Dex component
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

// TestValidateUpdate tests the Dex ValidateUpdate call for v1alpha1.Verrazzano
func TestValidateUpdate(t *testing.T) {
	disabled := false
	enabled := true
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with auth Dex disabled,
		// WHEN I call update the VZ CR to enable Dex component
		// THEN the update succeeds with no errors.
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			wantErr: false,
		},
		// GIVEN a VZ CR with Dex component enabled,
		// WHEN I call update the VZ CR to disable Dex component
		// THEN the update fails with an error.
		{
			name: "disable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Dex: &vzapi.DexComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with Dex component,
		// WHEN I call update with no change to the Dex component
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

// TestGetIngressNames tests the Dex GetIngressNames call
// GIVEN a Dex component
//
//	WHEN I call GetIngressNames
//	THEN the correct ingress name is returned
func TestGetIngressNames(t *testing.T) {
	ingressNames := NewComponent().GetIngressNames(nil)
	assert.True(t, len(ingressNames) == 1)
	assert.Equal(t, constants.DexIngress, ingressNames[0].Name)
	assert.Equal(t, ComponentNamespace, ingressNames[0].Namespace)
}

// TestDexCertificateNames tests the Dex GetCertificateNames call
// GIVEN a Dex component
// WHEN I call GetCertificateNames
// THEN the correct certificate name is returned
func TestDexCertificateNames(t *testing.T) {
	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Dex: &vzapi.DexComponent{
					Enabled: &enabled,
				},
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)
	names := NewComponent().GetCertificateNames(ctx)
	assert.Len(t, names, 1)
	assert.Equal(t, types.NamespacedName{Name: dexCertificateName, Namespace: ComponentNamespace}, names[0])
}

func getChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion: "v1",
			Name:       "hello",
			Version:    "0.1.0",
			AppVersion: "1.0",
		},
		Templates: []*chart.File{
			{Name: "templates/hello", Data: []byte("hello: world")},
		},
	}
}

func createRelease(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: "dex",
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart:   getChart(),
		Version: 1,
	}
}

func testActionConfigWithInstalledDex(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helmcli.CreateActionConfig(true, "dex", release.StatusDeployed, vzlog.DefaultLogger(), createRelease)
}

func testActionConfigWithUninstalledDex(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helmcli.CreateActionConfig(true, "dex", release.StatusUninstalled, vzlog.DefaultLogger(), createRelease)
}

// TestUninstallHelmChartInstalled tests the Dex Uninstall call
// GIVEN a dex component
//
//	WHEN I call Uninstall with the dex helm chart installed
//	THEN no error is returned
func TestUninstallHelmChartInstalled(t *testing.T) {
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(testActionConfigWithInstalledDex)

	k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace(constants.DexNamespace)
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestUninstallHelmChartNotInstalled tests the Dex Uninstall call
// GIVEN a Dex component
//
//	WHEN I call Uninstall with the Dex helm chart not installed
//	THEN no error is returned
func TestUninstallHelmChartNotInstalled(t *testing.T) {
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(testActionConfigWithUninstalledDex)

	k8sutil.GetCoreV1Func = common.MockGetCoreV1WithNamespace(constants.DexNamespace)
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}
