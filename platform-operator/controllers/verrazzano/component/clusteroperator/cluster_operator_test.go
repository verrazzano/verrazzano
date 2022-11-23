// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"os"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.ClusterOperator = &v1alpha1.ClusterOperatorComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	vzB1CR := &v1beta1.Verrazzano{}
	vzB1CROverrides := vzB1CR.DeepCopy()
	vzB1CROverrides.Spec.Components.ClusterOperator = &v1beta1.ClusterOperatorComponent{
		InstallOverrides: v1beta1.InstallOverrides{
			ValueOverrides: []v1beta1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		verrazzanoA1   *v1alpha1.Verrazzano
		verrazzanoB1   *v1beta1.Verrazzano
		expA1Overrides interface{}
		expB1Overrides interface{}
	}{
		{
			name:           "test no overrides",
			verrazzanoA1:   vzA1CR,
			verrazzanoB1:   vzB1CR,
			expA1Overrides: []v1alpha1.Overrides{},
			expB1Overrides: []v1beta1.Overrides{},
		},
		{
			name:           "test v1alpha1 enabled nil",
			verrazzanoA1:   vzA1CROverrides,
			verrazzanoB1:   vzB1CROverrides,
			expA1Overrides: vzA1CROverrides.Spec.Components.ClusterOperator.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.ClusterOperator.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCR()))
			asserts.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCRV1Beta1()))
		})
	}
}

// GIVEN a call to AppendOverrides
// WHEN  the env var for the cluster operator image is set
// THEN  the returned key/value pairs contains the image override
func TestAppendOverrides(t *testing.T) {
	customImage := "myreg.io/myrepo/v8o/verrazzano-cluster-operator-dev:local-20210707002801-b7449154"
	os.Setenv(constants.VerrazzanoClusterOperatorImageEnvVar, customImage)
	defer func() { os.Unsetenv(constants.VerrazzanoClusterOperatorImageEnvVar) }()

	kvs, err := AppendOverrides(nil, "", "", "", nil)
	asserts.NoError(t, err)
	asserts.Len(t, kvs, 1)
	asserts.Equal(t, "image", kvs[0].Key)
	asserts.Equal(t, customImage, kvs[0].Value)
}

// TestPostInstallUpgrade tests the PostInstallUpgrade creation of the RoleTemplate
func TestPostInstallUpgrade(t *testing.T) {
	clustOpComp := clusterOperatorComponent{}

	cli := fake.NewClientBuilder().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: vzconst.VerrazzanoClusterRancherName,
			},
		},
	).Build()
	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, nil, &v1beta1.Verrazzano{}, false))
	asserts.NoError(t, err)

	// Ensure the resource exists after postInstallUpgrade
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(rancher.GVKRoleTemplate)
	err = cli.Get(context.TODO(), types.NamespacedName{Name: vzconst.VerrazzanoClusterRancherName}, &resource)
	asserts.NoError(t, err)
}

// TestPostInstallUpgradeRancherDisabled tests the PostInstallUpgrade when Rancher is disabled
func TestPostInstallUpgradeRancherDisabled(t *testing.T) {
	clustOpComp := clusterOperatorComponent{}
	falseVal := false

	cli := fake.NewClientBuilder().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: vzconst.VerrazzanoClusterRancherName,
			},
		},
	).Build()
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Rancher: &v1alpha1.RancherComponent{
					Enabled: &falseVal,
				},
			},
		},
	}
	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, vz, nil, false))
	asserts.NoError(t, err)

	// Ensure the resource does not exist after postInstallUpgrade
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(rancher.GVKRoleTemplate)
	err = cli.Get(context.TODO(), types.NamespacedName{Name: vzconst.VerrazzanoClusterRancherName}, &resource)
	asserts.Error(t, err)
}
