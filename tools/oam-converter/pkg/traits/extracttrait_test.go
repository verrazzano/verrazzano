// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package traits

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Test cases for ExtractTrait function
func TestExtractTrait(t *testing.T) {
	// Test data: sample appMaps
	appMaps := []map[string]interface{}{
		{
			"apiVersion": "core.oam.dev/v1alpha2",
			"kind":       "ApplicationConfiguration",
			"metadata": map[string]interface{}{
				"name":      "hello-helidon",
				"namespace": "hello-helidon",
			},
			"spec": map[string]interface{}{
				"components": []interface{}{
					map[string]interface{}{
						"componentName": "hello-helidon-component",
						"traits": []interface{}{
							map[string]interface{}{
								"trait": map[string]interface{}{
									"apiVersion": "oam.verrazzano.io/v1alpha1",
									"kind":       "MetricsTrait",
									"spec": map[string]interface{}{
										"scraper": "verrazzano-system/vmi-system-prometheus-0",
									},
								},
							},
							map[string]interface{}{
								"trait": map[string]interface{}{
									"apiVersion": "oam.verrazzano.io/v1alpha1",
									"kind":       "IngressTrait",
									"metadata": map[string]interface{}{
										"name": "hello-helidon-ingress",
									},
									"spec": map[string]interface{}{
										"rules": []interface{}{
											map[string]interface{}{
												"paths": []interface{}{
													map[string]interface{}{
														"path":     "/greet",
														"pathType": "Prefix",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Call the function to test
	result, err := ExtractTrait(appMaps)

	// Assertions
	assert.NoError(t, err)

	expectedResult := []*types.ConversionComponents{
		{
			AppNamespace:  "hello-helidon",
			AppName:       "hello-helidon",
			ComponentName: "hello-helidon-component",
			IngressTrait: &vzapi.IngressTrait{
				TypeMeta: metav1.TypeMeta{
					Kind:       "IngressTrait",
					APIVersion: "oam.verrazzano.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "hello-helidon-ingress",
				},
				Spec: vzapi.IngressTraitSpec{
					Rules: []vzapi.IngressRule{
						{
							Destination: vzapi.IngressDestination{},
							Paths: []vzapi.IngressPath{
								{
									Path:     "/greet",
									PathType: "Prefix",
								},
							},
						},
					},
				},
			},
		},
	}
	assert.True(t, assert.Equal(t, expectedResult, result))
}
