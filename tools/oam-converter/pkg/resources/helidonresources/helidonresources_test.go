// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateIngressChildResourcesFromHelidon(t *testing.T) {
	// Create sample ConversionComponents, Gateway, and allHostsForTrait
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

	gateway := &vsapi.Gateway{}
	allHostsForTrait := []string{"example.com"}

	// Call the function with the sample inputs
	virtualServices, _, _, err := CreateIngressChildResourcesFromHelidon(conversionComponent, gateway, allHostsForTrait)

	// Verify the output
	assert.NoError(t, err, "Expected no error")
	assert.NotNil(t, virtualServices, "Expected non-nil virtualServices")
	// Check the length of the slices
	assert.NotEmpty(t, virtualServices, "Expected non-empty virtualServices")

}
