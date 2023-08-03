// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	reader "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/testdata"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	k8net "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateResources(t *testing.T) {

	appConf, err := reader.ReadFromYAMLTemplate("testdata/template/app_conf.yaml")
	if err != nil {
		t.Fatalf("Failed to read yaml file: %v", err)
	}

	spec, found, err := unstructured.NestedMap(appConf, "spec")
	if !found || err != nil {
		t.Fatalf("Spec component doesn't exist or not found in the specified type: %v", err)
	}

	appComponents, found, err := unstructured.NestedSlice(spec, "components")

	if !found || err != nil {
		t.Fatalf("app components doesn't exist or not found in the specified type: %v", err)
	}
	ingressData := make(map[string]interface{})
	for _, component := range appComponents {
		componentMap := component.(map[string]interface{})
		componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
		if ok && len(componentTraits) > 0 {
			for _, trait := range componentTraits {
				traitMap := trait.(map[string]interface{})
				ingressData, found, err = unstructured.NestedMap(traitMap, "trait")
				if !found || err != nil {
					t.Fatalf("ingress trait doesn't exist or not found in the specified type: %v", err)

				}

			}
		}
	}
	jsonData, err := json.Marshal(ingressData)
	if err != nil {
		t.Fatalf("error in marshalling ingress trait : %v", err)
	}
	ingressTrait := &vzapi.IngressTrait{}
	err = json.Unmarshal(jsonData, &ingressTrait)
	if err != nil {
		t.Fatalf("error in unmarshalling ingress trait : %v", err)
	}

	compConf, err := reader.ReadFromYAMLTemplate("testdata/template/helidon_workload.yaml")
	if err != nil {
		t.Fatalf("error in reading yaml file : %v", err)
	}

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

	conversionComponents := []*types.ConversionComponents{

		{
			AppName:         "myapp",
			ComponentName:   "mycomponent",
			AppNamespace:    "mynamespace",
			IngressTrait:    ingressTrait,
			Helidonworkload: helidonWorkload,
		},
	}
	ingress := k8net.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-ingress",
			Namespace: "verrazzano-system",
			Annotations: map[string]string{
				"external-dns.alpha.kubernetes.io/target": "test.nip.io",
			},
		},
	}
	cli := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&ingress).Build()
	// Call the method being tested
	kubeResources, err := CreateResources(cli, conversionComponents)

	// Check for errors
	assert.NoError(t, err, "Error occurred")

	// Assert that the returned kubeResources is not nil
	assert.NotNil(t, kubeResources, "kubeResources is nil")

	// Assert that the kubeResources fields are not empty or nil
	assert.NotEmpty(t, kubeResources.VirtualServices, "VirtualServices is empty")
	assert.NotNil(t, kubeResources.Gateway, "Gateway is nil")
}

func TestCreateChildResources(t *testing.T) {

	appConf, err := reader.ReadFromYAMLTemplate("testdata/template/app_conf.yaml")
	if err != nil {
		t.Fatalf("error in reading yaml file: %v", err)
	}

	spec, found, err := unstructured.NestedMap(appConf, "spec")
	if !found || err != nil {
		t.Fatalf("app spec doesn't exist or not found in specified type: %v", err)
	}

	appComponents, found, err := unstructured.NestedSlice(spec, "components")

	if !found || err != nil {
		t.Fatalf("app components doesn't exist or not found in specified type: %v", err)
	}
	ingressData := make(map[string]interface{})
	for _, component := range appComponents {
		componentMap := component.(map[string]interface{})
		componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
		if ok && len(componentTraits) > 0 {
			for _, trait := range componentTraits {
				traitMap := trait.(map[string]interface{})
				ingressData, found, err = unstructured.NestedMap(traitMap, "trait")
				if !found || err != nil {
					t.Fatalf("ingress trait doesn't exist or not found in specified type: %v", err)

				}

			}
		}
	}

	jsonData, err := json.Marshal(ingressData)
	if err != nil {
		t.Fatalf("error in marshalling ingress trait: %v", err)
	}
	ingressTrait := &vzapi.IngressTrait{}
	err = json.Unmarshal(jsonData, &ingressTrait)
	if err != nil {
		t.Fatalf("error in unmarshaling ingress trait: %v", err)
	}
	compConf, err := reader.ReadFromYAMLTemplate("testdata/template/helidon_workload.yaml")
	if err != nil {
		t.Fatalf("error in reading yaml file: %v", err)
	}
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

	// Test input data
	conversionComponent := &types.ConversionComponents{
		AppName:         "myapp",
		ComponentName:   "mycomponent",
		AppNamespace:    "mynamespace",
		IngressTrait:    ingressTrait,
		Helidonworkload: helidonWorkload,
	}

	// Mock the necessary dependencies (e.g., gateway, allHostsForTrait) if needed.
	gateway := &vsapi.Gateway{}
	allHostsForTrait := []string{"example.com"}
	cli := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	// Call the method being tested
	virtualServices, destinationRules, authzPolicies, err := createChildResources(cli, conversionComponent, gateway, allHostsForTrait)

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
