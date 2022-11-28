// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
// WHEN a VMC exists but no associated ClusterRoleBinding
// THEN no delete of a ClusterRoleBinding
func TestClusterOperatorEnabled(t *testing.T) {
	trueVal := true
	falseVal := false
	crA1 := &v1alpha1.Verrazzano{}
	crB1 := &v1beta1.Verrazzano{}

	crA1NilComp := crA1.DeepCopy()
	crA1NilComp.Spec.Components.ClusterOperator = nil
	crA1NilEnabled := crA1.DeepCopy()
	crA1NilEnabled.Spec.Components.ClusterOperator = &v1alpha1.ClusterOperatorComponent{Enabled: nil}
	crA1Enabled := crA1.DeepCopy()
	crA1Enabled.Spec.Components.ClusterOperator = &v1alpha1.ClusterOperatorComponent{Enabled: &trueVal}
	crA1Disabled := crA1.DeepCopy()
	crA1Disabled.Spec.Components.ClusterOperator = &v1alpha1.ClusterOperatorComponent{Enabled: &falseVal}

	crB1NilComp := crB1.DeepCopy()
	crB1NilComp.Spec.Components.ClusterOperator = nil
	crB1NilEnabled := crB1.DeepCopy()
	crB1NilEnabled.Spec.Components.ClusterOperator = &v1beta1.ClusterOperatorComponent{Enabled: nil}
	crB1Enabled := crB1.DeepCopy()
	crB1Enabled.Spec.Components.ClusterOperator = &v1beta1.ClusterOperatorComponent{Enabled: &trueVal}
	crB1Disabled := crB1.DeepCopy()
	crB1Disabled.Spec.Components.ClusterOperator = &v1beta1.ClusterOperatorComponent{Enabled: &falseVal}

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

// Test isReady when it's called with component context
func TestIsReady(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &v1alpha1.Verrazzano{}, nil, true)
	assert.False(t, NewComponent().IsReady(ctx))
}

// Test isReady when it's called with component context when dry run false
func TestIsReadyFalse(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &v1alpha1.Verrazzano{}, nil, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// TestPostInstall that the RoleTemplate gets created
func TestPostInstall(t *testing.T) {
	clustOpComp := clusterOperatorComponent{}

	cli := fake.NewClientBuilder().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: VerrazzanoClusterUserRoleName,
			},
		},
	).Build()
	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, &v1alpha1.Verrazzano{}, &v1beta1.Verrazzano{}, false))
	assert.NoError(t, err)
}

// TestPostUpgrade that the RoleTemplate gets created
func TestPostUpgrade(t *testing.T) {
	clustOpComp := clusterOperatorComponent{}

	cli := fake.NewClientBuilder().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: VerrazzanoClusterUserRoleName,
			},
		},
	).Build()
	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, &v1alpha1.Verrazzano{}, &v1beta1.Verrazzano{}, false))
	assert.NoError(t, err)
}
