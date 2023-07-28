// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateDestinationRuleFromHelidonWorkload(t *testing.T) {
	// Create a sample trait and rule
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "example-namespace",
		},
	}
	rule := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{
			HTTPCookie: &vzapi.IngressDestinationHTTPCookie{
				Name: "example-cookie",
				Path: "/example-path",
				TTL:  60,
			},
		},
	}

	// Create a sample helidon workload
	helidonWorkload := &unstructured.Unstructured{
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

	// Call the function being tested
	destinationRule, err := createDestinationRuleFromHelidonWorkload(trait, rule, "example-rule", helidonWorkload)

	// Assert that there is no error
	assert.NoError(t, err)

	// Assert that the destinationRule is not nil
	assert.NotNil(t, destinationRule)

	// Assert that the destinationRule has the correct APIVersion and Kind
	assert.Equal(t, "networking.istio.io/v1beta13", destinationRule.APIVersion)
	assert.Equal(t, "DestinationRule", destinationRule.Kind)

}
func TestMutateDestinationRuleFromHelidonWorkload(t *testing.T) {
	// Test data setup (same as before)
	destinationRule := &istioclient.DestinationRule{} // Create an empty destinationRule
	rule := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{
			HTTPCookie: &vzapi.IngressDestinationHTTPCookie{
				Name: "test-cookie",
				Path: "/test-path",
				TTL:  3600,
			},
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"istio-injection": "enabled",
			},
		},
	}
	helidonWorkload := &unstructured.Unstructured{
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

	// Call the function
	result, err := mutateDestinationRuleFromHelidonWorkload(destinationRule, rule, namespace, helidonWorkload)

	// Assert the result using testify/assert package
	assert.NoError(t, err, "Expected no error, but got one")
	assert.NotNil(t, result, "Expected a non-nil destinationRule, got nil")

	// Assert destinationRule content
	expectedHost := "hello-helidon-deployment" // Replace with the expected value
	assert.Equal(t, expectedHost, result.Spec.Host, "Unexpected Host value")

	expectedMode := istionet.ClientTLSSettings_ISTIO_MUTUAL
	assert.Equal(t, expectedMode, result.Spec.TrafficPolicy.Tls.Mode, "Unexpected TLS mode value")

}
