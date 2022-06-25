// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var enabled = true
var veleroEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Velero: &vzapi.VeleroComponent{
				Enabled: &enabled,
			},
		},
	},
}

//TestIsInstalled verifies component IsInstalled checks presence of the
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
			ctx := spi.NewFakeContext(tt.client, veleroEnabledCR, false)
			installed, err := NewComponent().IsInstalled(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.isInstalled, installed)
		})
	}
}

func TestInstallUpgrade(t *testing.T) {
	defer config.Set(config.Get())
	j := NewComponent()
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, veleroEnabledCR, false)
	err := j.Install(ctx)
	assert.NoError(t, err)
	err = j.Upgrade(ctx)
	assert.NoError(t, err)
	err = j.Reconcile(ctx)
	assert.NoError(t, err)
}

func TestGetMinVerrazzanoVersion(t *testing.T) {
	assert.Equal(t, constants.VerrazzanoVersion1_3_0, NewComponent().GetMinVerrazzanoVersion())
}

func TestGetDependencies(t *testing.T) {
	assert.Equal(t, []string{}, NewComponent().GetDependencies())
}

func TestGetName(t *testing.T) {
	j := NewComponent()
	assert.Equal(t, ComponentName, j.Name())
	assert.Equal(t, ComponentJSONName, j.GetJSONName())
}
