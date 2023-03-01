// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// TestGetOverrides tests if Thanos overrides are properly collected
// GIVEN a call to GetOverrides
// WHEN the VZ CR Thanos component has overrides
// THEN the overrides are returned from this function
func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.Thanos = &v1alpha1.ThanosComponent{
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
	vzB1CROverrides.Spec.Components.Thanos = &v1beta1.ThanosComponent{
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
			expA1Overrides: vzA1CROverrides.Spec.Components.Thanos.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.Thanos.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(tt.verrazzanoA1))
			asserts.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(tt.verrazzanoB1))
		})
	}
}
