// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestMutateVerrazzanoMonitoringNamespace tests the MutateVerrazzanoMonitoringNamespace function.
func TestMutateVerrazzanoMonitoringNamespace(t *testing.T) {
	// GIVEN a Verrazzano CR with Istio injection enabled
	//  WHEN we call the function to create the Verrazzano monitoring namespace struct
	//  THEN the struct has the expected labels, including the label with Istio injection enabled
	trueValue := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InjectionEnabled: &trueValue,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(nil, vz, nil, false)

	ns := GetVerrazzanoMonitoringNamespace()
	MutateVerrazzanoMonitoringNamespace(ctx, ns)
	assert.Equal(t, "enabled", ns.Labels[v8oconst.LabelIstioInjection])
	assert.Equal(t, constants.VerrazzanoMonitoringNamespace, ns.Labels[v8oconst.LabelVerrazzanoNamespace])
	assert.Equal(t, constants.VerrazzanoMonitoringNamespace, ns.Name)

	// GIVEN a Verrazzano CR with Istio injection disabled
	//  WHEN we call the function to create the Verrazzano monitoring namespace struct
	//  THEN the struct has the expected labels, excluding the Istio injection label
	falseValue := false
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InjectionEnabled: &falseValue,
				},
			},
		},
	}
	ctx = spi.NewFakeContext(nil, vz, nil, false)

	ns = GetVerrazzanoMonitoringNamespace()
	MutateVerrazzanoMonitoringNamespace(ctx, ns)
	assert.NotContains(t, ns.Labels, v8oconst.LabelIstioInjection)
	assert.Equal(t, constants.VerrazzanoMonitoringNamespace, ns.Labels[v8oconst.LabelVerrazzanoNamespace])
	assert.Equal(t, constants.VerrazzanoMonitoringNamespace, ns.Name)
}

// TestGetVerrazzanoMonitoringNamespace tests the GetVerrazzanoMonitoringNamespace function.
func TestGetVerrazzanoMonitoringNamespace(t *testing.T) {
	ns := GetVerrazzanoMonitoringNamespace()
	assert.Equal(t, constants.VerrazzanoMonitoringNamespace, ns.Name)
	assert.Nil(t, ns.Labels)
}

// TestEnsureMonitoringOperatorNamespace asserts the verrazzano-monitoring namespaces can be created
func TestEnsureMonitoringOperatorNamespace(t *testing.T) {
	// GIVEN a Verrazzano CR with Jaeger Component enabled,
	// WHEN we call the EnsureVerrazzanoMonitoringNamespace function,
	// THEN no error is returned.
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vzapi.Verrazzano{}, nil, false)
	err := EnsureVerrazzanoMonitoringNamespace(ctx)
	assert.NoError(t, err)
}
