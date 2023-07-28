// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateResources(t *testing.T) {
	// Test input data
	conversionComponents := []*types.ConversionComponents{

		{
			AppName:       "myapp",
			ComponentName: "mycomponent",
			AppNamespace:  "mynamespace",
			IngressTrait: &vzapi.IngressTrait{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-trait",
					Namespace: "mynamespace",
				},
				Spec: vzapi.IngressTraitSpec{
					Rules: []vzapi.IngressRule{
						{
							Paths: []vzapi.IngressPath{
								{Path: "/api/v1", PathType: "prefix"},
							},
						},
					},
				},
			},
			Helidonworkload: &unstructured.Unstructured{
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
			},
		},
	}

	// Call the method being tested
	kubeResources, err := CreateResources(conversionComponents)

	// Check for errors
	assert.NoError(t, err, "Error occurred")

	// Assert that the returned kubeResources is not nil
	assert.NotNil(t, kubeResources, "kubeResources is nil")

	// Assert that the kubeResources fields are not empty or nil
	assert.NotEmpty(t, kubeResources.VirtualServices, "VirtualServices is empty")
	assert.NotNil(t, kubeResources.Gateway, "Gateway is nil")
}

func TestCreateChildResources(t *testing.T) {
	// Test input data
	conversionComponent := &types.ConversionComponents{
		AppName:       "myapp",
		ComponentName: "mycomponent",
		AppNamespace:  "mynamespace",
		IngressTrait: &vzapi.IngressTrait{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trait",
				Namespace: "mynamespace",
			},
			Spec: vzapi.IngressTraitSpec{
				Rules: []vzapi.IngressRule{
					{
						Paths: []vzapi.IngressPath{
							{Path: "/api/v1", PathType: "prefix"},
						},
					},
				},
			},
		},
		Helidonworkload: &unstructured.Unstructured{
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
		},
	}

	// Mock the necessary dependencies (e.g., gateway, allHostsForTrait) if needed.
	gateway := &vsapi.Gateway{}
	allHostsForTrait := []string{"example.com"}
	// Call the method being tested
	virtualServices, destinationRules, authzPolicies, err := createChildResources(conversionComponent, gateway, allHostsForTrait)

	// Check for errors
	assert.NoError(t, err, "Error occurred")

	// Assert that the returned virtualServices, destinationRules, and authzPolicies are not nil
	assert.NotNil(t, virtualServices, "virtualServices is nil")
	assert.NotNil(t, destinationRules, "destinationRules is nil")
	assert.NotNil(t, authzPolicies, "authzPolicies is nil")

	// Assert that the returned slices are not empty
	assert.NotEmpty(t, virtualServices, "virtualServices is empty")
	assert.NotEmpty(t, destinationRules, "destinationRules is empty")
	assert.NotEmpty(t, authzPolicies, "authzPolicies is empty")

	expectedNumRules := len(conversionComponent.IngressTrait.Spec.Rules)
	assert.Len(t, virtualServices, expectedNumRules, "unexpected number of VirtualServices")
	assert.Len(t, destinationRules, expectedNumRules, "unexpected number of DestinationRules")
	assert.Len(t, authzPolicies, expectedNumRules, "unexpected number of AuthorizationPolicies")
}
