// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestVZContainsResource tests if the VZ CR contains an object
// GIVEN a Verrazzano, and an object
// WHEN the method is called
// THEN return if the object is found in the Verrazzano
func TestVZContainsResource(t *testing.T) {
	tests := []struct {
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
				Status: vzapi.VerrazzanoStatus{
					Components: vzapi.ComponentStatusMap{
						"prometheus-operator": &vzapi.ComponentStatusDetails{
							Name:                     "prometheus-operator",
							LastReconciledGeneration: 0,
							ReconcilingGeneration:    0,
						},
					},
					State: vzapi.VzStateReady,
				},
			},
			object: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind: configMapKind,
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
				Status: vzapi.VerrazzanoStatus{
					Components: vzapi.ComponentStatusMap{
						"prometheus-operator": &vzapi.ComponentStatusDetails{
							Name:                     "prometheus-operator",
							LastReconciledGeneration: 0,
							ReconcilingGeneration:    0,
						},
					},
					State: vzapi.VzStateReady,
				},
			},
			object: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind: secretKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sec",
				},
			},
			expect: true,
		},
	}
	a := asserts.New(t)
	config.SetDefaultBomFilePath(testBomFilePath)
	for _, tt := range tests {
		mocker := gomock.NewController(t)
		mockCli := mocks.NewMockClient(mocker)
		if tt.expect {
			mockStatus := mocks.NewMockStatusWriter(mocker)
			mockStatus.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)
			mockStatus.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil())).Return(nil)
			mockCli.EXPECT().Status().Return(mockStatus).AnyTimes()
		}
		r := newVerrazzanoReconciler(mockCli)
		t.Run(tt.name, func(t *testing.T) {
			context := spi.NewFakeContext(mockCli, tt.vz, false)
			a.Equal(tt.expect, r.vzContainsResource(context, tt.object))
		})
	}
}
