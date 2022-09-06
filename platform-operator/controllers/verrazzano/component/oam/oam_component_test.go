// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = rbacv1.AddToScheme(testScheme)
}

func TestValidateUpdate(t *testing.T) {
	disabled := false
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
						OAM: &vzapi.OAMComponent{
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
						OAM: &vzapi.OAMComponent{
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

func TestValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						OAM: &v1beta1.OAMComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						OAM: &v1beta1.OAMComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
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

// TestPreUpgrade tests the OAM PreUpgrade call
// GIVEN an OAM component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../thirdparty"
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), nil, nil, false))
	assert.NoError(t, err)
}

// TestPostInstall tests post-install processing
// GIVEN an OAM component
// WHEN I call PostInstall with defaults
// THEN no error is returned and the expected cluster roles exists
func TestPostInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := NewComponent().PostInstall(spi.NewFakeContext(client, nil, nil, false))
	assert.NoError(t, err)

	var clusterRole rbacv1.ClusterRole
	err = client.Get(context.TODO(), types.NamespacedName{Name: pvcClusterRoleName}, &clusterRole)
	assert.NoError(t, err)
}

// TestPostUpgrade tests post-upgrade processing
// GIVEN an OAM component
// WHEN I call PostUpgrade with defaults
// THEN no error is returned and the expected cluster roles exists
func TestPostUpgrade(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := NewComponent().PostUpgrade(spi.NewFakeContext(client, nil, nil, false))
	assert.NoError(t, err)

	var clusterRole rbacv1.ClusterRole
	err = client.Get(context.TODO(), types.NamespacedName{Name: pvcClusterRoleName}, &clusterRole)
	assert.NoError(t, err)
}
