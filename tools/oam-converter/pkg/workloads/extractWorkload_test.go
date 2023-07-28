// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloads

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestExtractWorkload(t *testing.T) {
	// Test data: components array
	components := []map[string]interface{}{
		{
			"apiVersion": "core.oam.dev/v1alpha2",
			"kind":       "Component",
			"metadata": map[string]interface{}{
				"name": "hello-helidon-component",
			},
			"spec": map[string]interface{}{
				"workload": map[string]interface{}{
					"apiVersion": "oam.verrazzano.io/v1alpha1",
					"kind":       "VerrazzanoHelidonWorkload",
					"metadata": map[string]interface{}{
						"name": "hello-helidon-workload",
						"labels": map[string]interface{}{
							"app":     "hello-helidon",
							"version": "v1",
						},
					},
					"spec": map[string]interface{}{
						"deploymentTemplate": map[string]interface{}{
							"metadata": map[string]interface{}{
								"name": "hello-helidon-deployment",
							},
							"podSpec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "hello-helidon-container",
										"image": "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20230126194830-31cd41f",
										"ports": []interface{}{
											map[string]interface{}{
												"containerPort": int64(8080),
												"name":          "http",
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

	// Test data: conversionComponents array
	conversionComponents := []*types.ConversionComponents{

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

	// Call the function to test
	result, err := ExtractWorkload(components, conversionComponents)

	// Assertions
	assert.NoError(t, err)

	expectedHelidonWorkload := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "oam.verrazzano.io/v1alpha1",
			"kind":       "VerrazzanoHelidonWorkload",
			"metadata": map[string]interface{}{
				"name": "hello-helidon-workload",
				"labels": map[string]interface{}{
					"app":     "hello-helidon",
					"version": "v1",
				},
			},
			"spec": map[string]interface{}{
				"deploymentTemplate": map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "hello-helidon-deployment",
					},
					"podSpec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "hello-helidon-container",
								"image": "ghcr.io/verrazzano/example-helidon-greet-app-v1:1.0.0-1-20230126194830-31cd41f",
								"ports": []interface{}{
									map[string]interface{}{
										"containerPort": int64(8080),
										"name":          "http",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedHelidonWorkload, result[0].Helidonworkload)

}
