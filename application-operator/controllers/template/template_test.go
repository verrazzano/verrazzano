package template

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	client := getTestClient()
	expectedOutput := `data:
  template:
    - value1: "This is the string associated with value1"
    - value2: "This is the string associated with value2"
    - secret: "totallySecretValue"
`
	tp := NewProcessor("testNamespace", client, "../../test/templates/template_with_kube_resource_values.yaml")
	output, err := tp.processTemplate(nil)
	assert.NoError(t, err, "error executing template processing for config map values")
	assert.Equal(t, expectedOutput, output)
}

// TestPopulateFromStructs tests whether a template can be populated with the values from a provided structs
// GIVEN that a template processor referencing multiple structs
// WHEN the processor executes
// THEN the resulting string should be populated with values from the structs provided as inputs
func TestPopulateFromStructs(t *testing.T) {
	client := getTestClient()
	inputs := map[string]interface{}{
		"workload": Workload{Name: "Workload1"},
		"profile":  Profile{Name: "Profile1"},
	}
	expectedOutput := `data:
  template:
    - workload: "Workload1"
    - profile: "Profile1"
`
	tp := NewProcessor("testNamespace", client, "../../test/templates/template_with_struct_values.yaml")
	output, err := tp.processTemplate(inputs)
	assert.NoError(t, err, "error executing template processing for config map values")
	assert.Equal(t, expectedOutput, output)
}

// TestPopulateFromStructsAndResources tests whether a template can be populated with the values from provided structs and kube resources
// GIVEN that a template processor referencing multiple sources
// WHEN the processor executes
// THEN the resulting string should be populated with values from the sources indicated
func TestPopulateFromStructsAndResources(t *testing.T) {
	client := getTestClient()
	inputs := map[string]interface{}{
		"workload": Workload{Name: "Workload1"},
		"profile":  Profile{Name: "Profile1"},
	}
	expectedOutput := `data:
  template:
    - value1: "This is the string associated with value1"
    - value2: "This is the string associated with value2"
    - secret: "totallySecretValue"
    - workload: "Workload1"
    - profile: "Profile1"
`
	tp := NewProcessor("testNamespace", client, "../../test/templates/template_with_resources_and_struct_values.yaml")
	output, err := tp.processTemplate(inputs)
	assert.NoError(t, err, "error executing template processing for config map values")
	assert.Equal(t, expectedOutput, output)
}

// getTestClient returns a controller/kube client with references to simple secret and configmap
func getTestClient() client.Client {
	client := fake.NewFakeClientWithScheme(newScheme(),
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "someSecret",
				Namespace: "testNamespace",
			},
			Data: map[string][]byte{
				"secret1": []byte("totallySecretValue"),
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testConfigMap",
				Namespace: "testNamespace",
			},
			Data: map[string]string{
				"value1": "This is the string associated with value1",
				"value2": "This is the string associated with value2",
			},
		})
	return client
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	corev1.SchemeBuilder.AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{
		Version: "v1",
	}, &corev1.Secret{}, &corev1.ConfigMap{})

	return scheme
}
