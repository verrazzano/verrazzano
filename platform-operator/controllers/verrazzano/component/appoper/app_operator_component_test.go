// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"testing"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	v1alpha12 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profilesRelativePath = "../../../../manifests/profiles"
	relativeRootDir      = "../../../../../"
)

var crEnabled = v1alpha1.Verrazzano{
	Spec: v1alpha1.VerrazzanoSpec{
		Components: v1alpha1.ComponentSpec{
			ApplicationOperator: &v1alpha1.ApplicationOperatorComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
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
	return helm.CreateActionConfig(true, "verrazzano-application-operator", release.StatusDeployed, vzlog.DefaultLogger(), createRelease)
}

func TestPreInstall(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: relativeRootDir})
	client := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	assert.NoError(t, NewComponent().PreInstall(ctx))
	metricsBindingCRD := &apiextensionsv1.CustomResourceDefinition{}
	// Metrics Binding CRD should exist after PreInstall
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: "metricsbindings.app.verrazzano.io"}, metricsBindingCRD))
}

// TestPreUpgrade tests the PreUpgrade function
// GIVEN a call to PreUpgrade
// WHEN the Helm chart is deployed and CRDs exist
// THEN no error during PreUpgrade
func TestPreUpgrade(t *testing.T) {
	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: relativeRootDir})
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallation)

	a := NewComponent()
	client := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(append(testTraitObjects(), testWorkloadDefinitionObjects()...)...).
		Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	err := a.PreUpgrade(ctx)
	assert.NoError(t, err)
}

// TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
// WHEN a VMC exists but no associated ClusterRoleBinding
// THEN no delete of a ClusterRoleBinding
func TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding(t *testing.T) {
	clusterName := "managed1"
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				DNS: &v1alpha1.DNSComponent{
					OCI: &v1alpha1.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
		&v1alpha12.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		}).Build()
	err := NewComponent().PostUpgrade(spi.NewFakeContext(fakeClient, vz, nil, false))
	assert.NoError(t, err)
}

// TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
// WHEN the VMC CRD does not exist
// THEN no error is returned
func TestAppOperatorVMCMissing(t *testing.T) {
	vz := &v1alpha1.Verrazzano{}
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(&meta.NoKindMatchError{})

	err := NewComponent().PostUpgrade(spi.NewFakeContext(mockClient, vz, nil, false))
	assert.NoError(t, err)
}

// TestAppOperatorPostUpgradeDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
// WHEN a VMC exists with an associated ClusterRoleBinding
// THEN successful delete of the ClusterRoleBinding
func TestAppOperatorPostUpgradeDeleteClusterRoleBinding(t *testing.T) {
	clusterName := "managed1"
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				DNS: &v1alpha1.DNSComponent{
					OCI: &v1alpha1.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
		&v1alpha12.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("verrazzano-cluster-%s", clusterName),
			},
			Subjects: nil,
			RoleRef:  rbacv1.RoleRef{},
		}).Build()
	err := NewComponent().PostUpgrade(spi.NewFakeContext(fakeClient, vz, nil, false))
	assert.Nil(t, err)

	// Verify the ClusterRolebinding was deleted
	clusterRoleBinding := rbacv1.ClusterRoleBinding{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("verrazzano-cluster-%s", clusterName)}, &clusterRoleBinding)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1alpha12.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = oamv1alpha1.AddToScheme(scheme)
	_ = oamv1alpha2.SchemeBuilder.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	return scheme
}

// TestIsEnabledNilApplicationOperator tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The ApplicationOperator component is nil
// THEN true is returned
func TestIsEnabledNilApplicationOperator(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.ApplicationOperator = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The ApplicationOperator component is nil
// THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The ApplicationOperator component enabled is nil
// THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.ApplicationOperator.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The ApplicationOperator component is explicitly enabled
// THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.ApplicationOperator.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
// WHEN The ApplicationOperator component is explicitly disabled
// THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.ApplicationOperator.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}

func Test_applicationOperatorComponent_ValidateUpdate(t *testing.T) {
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
						ApplicationOperator: &v1alpha1.ApplicationOperatorComponent{
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
						ApplicationOperator: &v1alpha1.ApplicationOperatorComponent{
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

			v1beta1New := &v1beta1.Verrazzano{}
			v1beta1Old := &v1beta1.Verrazzano{}
			err := tt.new.ConvertTo(v1beta1New)
			assert.NoError(t, err)
			err = tt.old.ConvertTo(v1beta1Old)
			assert.NoError(t, err)
			if err := c.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
