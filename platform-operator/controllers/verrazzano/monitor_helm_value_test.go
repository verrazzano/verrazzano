// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestVZContainsResource tests if the VZ CR contains an object
// GIVEN a Verrazzano, and an object
// WHEN the method is called
// THEN return if the object is found in the Verrazzano
func TestVZContainsResource(t *testing.T) {
	var tests = []struct {
		name   string
		vz     *vzapi.Verrazzano
		object client.Object
		expect bool
	}{
		{
			name:   "test not found",
			vz:     &vzapi.Verrazzano{},
			object: &v1.ConfigMap{},
			expect: false,
		},
		{
			name: "test found configmap",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							HelmValueOverrides: vzapi.HelmValueOverrides{
								ValueOverrides: []vzapi.Overrides{
									{
										ConfigMapRef: &v1.ConfigMapKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-cm",
											},
											Optional: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			object: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind: configMap,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cm",
				},
			},
			expect: true,
		},
		{
			name: "test found secret",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							HelmValueOverrides: vzapi.HelmValueOverrides{
								ValueOverrides: []vzapi.Overrides{
									{
										SecretRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "test-sec",
											},
											Optional: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			object: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind: secret,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sec",
				},
			},
			expect: true,
		},
	}
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	r := newVerrazzanoReconciler(c)
	a := asserts.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := spi.NewFakeContext(c, tt.vz, false)
			a.Equal(r.vzContainsResource(context, tt.object), tt.expect)
		})
	}
}
