// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"encoding/json"
	asserts "github.com/stretchr/testify/assert"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type Workload struct {
	Name string
}
type Profile struct {
	Name string
}

// TestPopulateFromKubeResources tests whether a template can be populated with the values from a configmap and secret
// GIVEN that a template processor referencing a configmap and secret in the same namespace
// WHEN the processor executes
// THEN the resulting string should be populated with values from the config map and secret
func TestPopulateFromKubeResources(t *testing.T) {
	c := getTestClient()
	templateText := `data:
  template:
    - value1: "{{configmap "testNamespace" "testConfigMap" "value1"}}"
    - value2: "{{configmap "testNamespace" "testConfigMap" "value2"}}"
    - secret: "{{secret "testNamespace" "someSecret" "secret1"}}"
`
	expectedOutput := `data:
  template:
    - value1: "This is the string associated with value1"
    - value2: "This is the string associated with value2"
    - secret: "totallySecretValue"
`
	tp := NewProcessor(c, templateText)
	output, err := tp.Process(nil)
	asserts.NoError(t, err, "error executing template processing for config map values")
	asserts.Equal(t, expectedOutput, output)
}

// TestPopulateFromStructs tests whether a template can be populated with the values from a provided structs
// GIVEN that a template processor referencing multiple structs
// WHEN the processor executes
// THEN the resulting string should be populated with values from the structs provided as inputs
func TestPopulateFromStructs(t *testing.T) {
	c := getTestClient()
	inputs := map[string]interface{}{
		"workload": Workload{Name: "Workload1"},
		"profile":  Profile{Name: "Profile1"},
	}
	templateText := `data:
  template:
    - workload: "{{.workload.Name}}"
    - profile: "{{.profile.Name}}"
`
	expectedOutput := `data:
  template:
    - workload: "Workload1"
    - profile: "Profile1"
`
	tp := NewProcessor(c, templateText)
	output, err := tp.Process(inputs)
	asserts.NoError(t, err, "error executing template processing for config map values")
	asserts.Equal(t, expectedOutput, output)
}

// TestPopulateFromStructsAndResources tests whether a template can be populated with the values from provided structs and kube resources
// GIVEN that a template processor referencing multiple sources
// WHEN the processor executes
// THEN the resulting string should be populated with values from the sources indicated
func TestPopulateFromStructsAndResources(t *testing.T) {
	c := getTestClient()
	inputs := map[string]interface{}{
		"workload": Workload{Name: "Workload1"},
		"profile":  Profile{Name: "Profile1"},
	}
	templateText := `data:
  template:
    - value1: "{{configmap "testNamespace" "testConfigMap" "value1"}}"
    - value2: "{{configmap "testNamespace" "testConfigMap" "value2"}}"
    - secret: "{{secret "testNamespace" "someSecret" "secret1"}}"
    - workload: "{{.workload.Name}}"
    - profile: "{{.profile.Name}}"
`
	expectedOutput := `data:
  template:
    - value1: "This is the string associated with value1"
    - value2: "This is the string associated with value2"
    - secret: "totallySecretValue"
    - workload: "Workload1"
    - profile: "Profile1"
`
	tp := NewProcessor(c, templateText)
	output, err := tp.Process(inputs)
	asserts.NoError(t, err, "error executing template processing for config map values")
	asserts.Equal(t, expectedOutput, output)
}

// TestPopulateFromUnstructured tests populating a template from an Unstructured input
// GIVEN a template that references a nested value within an Unstructured
// WHEN the processor executes
// THEN the resulting string should be populated from the Unstructured's values
func TestPopulateFromUnstructured(t *testing.T) {
	assert := asserts.New(t)
	c := getTestClient()
	workload, err := convertToUnstructured(
		k8sapps.Deployment{
			ObjectMeta: k8smeta.ObjectMeta{
				Namespace: "test-namespace-name",
				Name:      "test-workload-name",
			}})
	assert.NoError(err, "error converting resource to unstructured")
	inputs := map[string]interface{}{
		"workload": workload.Object,
	}
	templateText := `<{{.workload.metadata.namespace}}/{{.workload.metadata.name}}>`
	expectedOutput := `<test-namespace-name/test-workload-name>`
	tp := NewProcessor(c, templateText)
	actualOutput, err := tp.Process(inputs)
	assert.NoError(err, "error processing template")
	assert.Equal(expectedOutput, actualOutput, "expected template to be populated correctly from unstructured")
}

// TestPopulateFromGet tests populating a template from values obtained from a fetched resource
// GIVEN a template that references a function to get a resource
// WHEN the processor executes
// THEN the resulting string should be populated from the resources values
func TestPopulateFromGet(t *testing.T) {
	assert := asserts.New(t)
	c := getTestClient()
	templateText := `{{$cm := get "v1" "ConfigMap" "testNamespace" "testConfigMap"}}<{{$cm.metadata.name}}>`
	expectedOutput := `<testConfigMap>`
	tp := NewProcessor(c, templateText)
	actualOutput, err := tp.Process(nil)
	assert.NoError(err, "error processing template")
	assert.Equal(expectedOutput, actualOutput, "expected template to be populated correctly from unstructured")
}

// getTestClient returns a controller/kube client with references to simple secret and configmap
func getTestClient() client.Client {
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
		&k8score.Secret{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "someSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"secret1": []byte("totallySecretValue"),
			},
		},
		&k8score.ConfigMap{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "testConfigMap",
				Namespace: "testNamespace",
			},
			Data: map[string]string{
				"value1": "This is the string associated with value1",
				"value2": "This is the string associated with value2",
			},
		}).Build()
	return c
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	k8score.SchemeBuilder.AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &k8score.Secret{}, &k8score.ConfigMap{})

	return scheme
}

// convertToUnstructured converts an object to an Unstructured version
// object - The object to convert to Unstructured
func convertToUnstructured(object interface{}) (unstructured.Unstructured, error) {
	jbytes, err := json.Marshal(object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	var u map[string]interface{}
	json.Unmarshal(jbytes, &u)
	return unstructured.Unstructured{Object: u}, nil
}
