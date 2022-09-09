// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var trueValue = true
var falseValue = false

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
	tests := []struct {
		name              string
		cr                *vzapi.Verrazzano
		existingNamespace *corev1.Namespace
		expectedLabels    map[string]string
	}{
		{
			// GIVEN a default Verrazzano CR and no verrazzano-monitoring namespace,
			// WHEN we call the EnsureVerrazzanoMonitoringNamespace function,
			// THEN no error is returned and the namespace is created with the required labels.
			"Default VZ CR with no namespace",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							InjectionEnabled: &trueValue,
						},
					},
				},
			},
			nil,
			map[string]string{
				v8oconst.LabelVerrazzanoNamespace: constants.VerrazzanoMonitoringNamespace,
				v8oconst.LabelIstioInjection:      "enabled",
			},
		},
		{
			// GIVEN a default Verrazzano CR having prior verrazzano-monitoring namespace with no labels,
			// WHEN we call the EnsureVerrazzanoMonitoringNamespace function,
			// THEN no error is returned and the namespace is created with the required labels.
			"Default VZ CR with already existing namespace",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							InjectionEnabled: &trueValue,
						},
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.VerrazzanoMonitoringNamespace,
				},
			},
			map[string]string{
				v8oconst.LabelVerrazzanoNamespace: constants.VerrazzanoMonitoringNamespace,
				v8oconst.LabelIstioInjection:      "enabled",
			},
		},
		{
			// GIVEN a Verrazzano CR with istio disabled, and prior verrazzano-monitoring namespace with no labels,
			// WHEN we call the EnsureVerrazzanoMonitoringNamespace function,
			// THEN no error is returned and the existing namespace is mutated without the istio label
			"Istio disabled VZ CR with already existing namespace",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.VerrazzanoMonitoringNamespace,
				},
			},
			map[string]string{
				v8oconst.LabelVerrazzanoNamespace: constants.VerrazzanoMonitoringNamespace,
			},
		},
		{
			// GIVEN a Verrazzano CR with istio disabled, and prior verrazzano-monitoring namespace with istio labels,
			// WHEN we call the EnsureVerrazzanoMonitoringNamespace function,
			// THEN no error is returned and the existing namespace is mutated without the istio label
			"Istio disabled VZ CR with already existing namespace with istio labels",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.VerrazzanoMonitoringNamespace,
					Labels: map[string]string{
						v8oconst.LabelIstioInjection:      "enabled",
						v8oconst.LabelVerrazzanoNamespace: constants.VerrazzanoMonitoringNamespace,
					},
				},
			},
			map[string]string{
				v8oconst.LabelVerrazzanoNamespace: constants.VerrazzanoMonitoringNamespace,
				v8oconst.LabelIstioInjection:      "enabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var existingNamespaces []runtime.Object
			if tt.existingNamespace != nil {
				existingNamespaces = append(existingNamespaces, tt.existingNamespace)
			}
			fakeclient := fake.NewClientBuilder().WithScheme(testScheme).WithRuntimeObjects(existingNamespaces...).Build()
			ctx := spi.NewFakeContext(fakeclient, tt.cr, nil, false)
			err := EnsureVerrazzanoMonitoringNamespace(ctx)
			assert.NoError(t, err)
			actualNamespace := corev1.Namespace{}
			err = fakeclient.Get(context.TODO(),
				types.NamespacedName{
					Name: constants.VerrazzanoMonitoringNamespace,
				}, &actualNamespace)
			assert.NoError(t, err)
			for labelKey, labelValue := range tt.expectedLabels {
				assert.Equal(t, labelValue, actualNamespace.Labels[labelKey])
			}
			// check the name matches verrazzano-monitoring
			assert.Equal(t, constants.VerrazzanoMonitoringNamespace, actualNamespace.Name)
			// check there are no additional labels
			assert.Equal(t, len(tt.expectedLabels), len(actualNamespace.Labels))
		})
	}

}
