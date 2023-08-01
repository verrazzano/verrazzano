// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package workloads

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	reader "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/testdata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateDestinationRuleFromWorkload(t *testing.T) {
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
	compMaps := []map[string]interface{}{}

	compConf, err := reader.ReadFromYAMLTemplate("testdata/template/helidon_workload.yaml")
	if err != nil {
		t.Fatalf("error in reading yaml file : %v", err)
	}
	compMaps = append(compMaps, compConf)
	compSpec, found, err := unstructured.NestedMap(compConf, "spec")
	if !found || err != nil {
		t.Fatalf("component spec doesn't exist or not found in specified type: %v", err)
	}
	compWorkload, found, err := unstructured.NestedMap(compSpec, "workload")
	if !found || err != nil {
		t.Fatalf("component workload doesn't exist or not found in specified type: %v", err)
	}
	helidonWorkload := &unstructured.Unstructured{
		Object: compWorkload,
	}
	service := &corev1.Service{}

	// Call the function being tested
	destinationRule, err := createDestinationRuleFromWorkload(trait, rule, "example-rule", helidonWorkload, service)

	// Assert that there is no error
	assert.NoError(t, err)

	// Assert that the destinationrule is not nil
	assert.NotNil(t, destinationRule)

	// Assert that the destinationrule has the correct APIVersion and Kind
	assert.Equal(t, "networking.istio.io/v1beta13", destinationRule.APIVersion)
	assert.Equal(t, "DestinationRule", destinationRule.Kind)

}
