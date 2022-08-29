// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"testing"

	"github.com/Jeffail/gabs/v2"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test_updateStringMap tests metrics trait utility function updateStringMap
func Test_updateStringMap(t *testing.T) {
	assert := asserts.New(t)
	var input map[string]string
	var output map[string]string

	// GIVEN a nil input map
	// WHEN a new name value pair are added
	// THEN verify a map is returned containing the new name value pair.
	input = nil
	output = updateStringMap(input, "test-name-1", "test-value-1")
	assert.Len(output, 1)
	assert.Equal("test-value-1", output["test-name-1"])

	// GIVEN an empty input map
	// WHEN a new name value pair are added
	// THEN verify a map is returned containing the new name value pair.
	input = map[string]string{}
	output = updateStringMap(input, "test-name-1", "test-value-1")
	assert.Len(output, 1)
	assert.Equal("test-value-1", output["test-name-1"])

	// GIVEN an map with an existing name/value pair
	// WHEN a new value is set for an existing name
	// THEN verify a map contains the new value
	input = map[string]string{"test-name-1": "test-value-1"}
	output = updateStringMap(input, "test-name-1", "test-value-2")
	assert.Len(output, 1)
	assert.Equal("test-value-2", output["test-name-1"])

	// GIVEN an map with an existing name/value pair
	// WHEN a new name and value is set
	// THEN verify a map contains both the old and the new pairs
	input = map[string]string{"test-name-1": "test-value-1"}
	output = updateStringMap(input, "test-name-2", "test-value-2")
	assert.Len(output, 2)
	assert.Equal("test-value-1", output["test-name-1"])
	assert.Equal("test-value-2", output["test-name-2"])
}

// Test_copyStringMapEntries tests metrics trait utility function copyStringMapEntries
func Test_copyStringMapEntries(t *testing.T) {
	assert := asserts.New(t)
	var source map[string]string
	var target map[string]string
	var output map[string]string

	// GIVEN nil source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the target map is empty
	source = nil
	target = nil
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN empty source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the target map is empty
	source = map[string]string{}
	target = map[string]string{}
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN empty source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the output and target map have two entries
	source = map[string]string{"test-name-1": "test-value-1"}
	target = map[string]string{"test-name-2": "test-value-2"}
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Equal("test-value-1", output["test-name-1"])
	assert.Equal("test-value-2", output["test-name-2"])
	assert.Len(output, 2)
	assert.Len(source, 1)
	assert.Len(target, 2)
}

// Test_getNamespaceFromObjectMetaOrDefault tests metrics trait utility function getNamespaceFromObjectMetaOrDefault
func Test_getNamespaceFromObjectMetaOrDefault(t *testing.T) {
	assert := asserts.New(t)
	var meta metav1.ObjectMeta
	var name string

	// GIVEN metadata with a blank namespace name
	// WHEN the namespace name is retrieved
	// THEN verify the "default" namespace name is returned
	name = getNamespaceFromObjectMetaOrDefault(meta)
	assert.Equal("default", name)

	// GIVEN metadata with a non-blank namespace name
	// WHEN the namespace name is retrieved
	// THEN verify the correct namespace name is returned
	meta = metav1.ObjectMeta{Namespace: "test-namespace-1"}
	name = getNamespaceFromObjectMetaOrDefault(meta)
	assert.Equal("test-namespace-1", name)
}

// Test_parseYAMLString tests metrics trait utility function parseYAMLString
// func parseYAMLString(s string) (*gabs.Container, error) {
func Test_parseYAMLString(t *testing.T) {
	assert := asserts.New(t)
	var cont *gabs.Container
	var str string
	var err error

	// GIVEN an empty yaml string
	// WHEN the yaml string is parsed
	// THEN verify that the unstructured objects are empty
	str = ""
	cont, err = parseYAMLString(str)
	assert.NoError(err)
	assert.Equal(nil, cont.Data())

	// GIVEN an invalid yaml string
	// WHEN the yaml string is parsed
	// THEN verify that an error is returned
	str = ":"
	cont, err = parseYAMLString(str)
	assert.Error(err)

	// GIVEN an simple yaml string
	// WHEN the yaml string is parsed
	// THEN verify that the unstructured objects contain the correct data
	str = "test-name-1: test-value-1"
	cont, err = parseYAMLString(str)
	assert.NoError(err)
	assert.Equal("test-value-1", cont.Path("test-name-1").Data().(string))
}

// Test_writeYAMLString tests metrics trait utility function writeYAMLString
func Test_writeYAMLString(t *testing.T) {
	assert := asserts.New(t)
	var str string
	var err error

	// GIVEN an simple yaml container
	// WHEN the yaml container is written to a string
	// THEN verify that the string is correct
	var cont = gabs.New()
	cont.Set("test-value-1", "test-name-1")
	str, err = writeYAMLString(cont)
	assert.NoError(err)
	assert.Equal("test-name-1: test-value-1\n", str)
}

// Test_mergeTemplateWithContext tests metrics trait utility function mergeTemplateWithContext
func Test_mergeTemplateWithContext(t *testing.T) {
	assert := asserts.New(t)
	var input string
	var output string
	var context map[string]string

	// GIVEN an empty template and nil context
	// WHEN the template and context are merged
	// THEN verify that the result is an empty string
	input = ""
	context = nil
	output = mergeTemplateWithContext(input, context)
	assert.Equal("", output)

	// GIVEN an template with no placeholders and nil context
	// WHEN the template and context are merged
	// THEN verify that the result is the same as the input
	input = "no-place-holders"
	context = nil
	output = mergeTemplateWithContext(input, context)
	assert.Equal("no-place-holders", output)

	// GIVEN an template with duplicate placeholder and a newline
	// WHEN the context contains a value for the placeholder
	// THEN verify that the output has the value twice with a newline separating them
	input = "{{template-name-1}}\n{{template-name-1}}"
	context = map[string]string{"{{template-name-1}}": "template-value-2"}
	output = mergeTemplateWithContext(input, context)
	assert.Equal("template-value-2\ntemplate-value-2", output)
}

// TestGetSupportedWorkloadType tests metrics trait utility function GetSupportedWorkloadType
func TestGetSupportedWorkloadType(t *testing.T) {
	assert := asserts.New(t)
	var apiVerKind string
	var workloadType string

	// GIVEN an api version weblogic.oracle/v8 and Kind Domain
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadtype is weblogic
	apiVerKind = "weblogic.oracle/v8.Domain"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Equal(constants.WorkloadTypeWeblogic, workloadType)

	// GIVEN an api version coherence.oracle.com/v1 and Kind Coherence
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadType is coherence
	apiVerKind = "coherence.oracle.com/v1.Coherence"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Equal(constants.WorkloadTypeCoherence, workloadType)

	// GIVEN an api version oam.verrazzano.io/v1alpha1 and Kind VerrazzanoHelidonWorkload
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadType is generic
	apiVerKind = "oam.verrazzano.io/v1alpha1.VerrazzanoHelidonWorkload"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Equal(constants.WorkloadTypeGeneric, workloadType)

	// GIVEN an api version core.oam.dev/v1alpha2 and Kind ContainerizedWorkload
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadType is generic
	apiVerKind = "core.oam.dev/v1alpha2.ContainerizedWorkload"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Equal(constants.WorkloadTypeGeneric, workloadType)

	// GIVEN an api version apps/v1 and Kind Deployment
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadType is generic
	apiVerKind = "apps/v1.Deployment"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Equal(constants.WorkloadTypeGeneric, workloadType)

	// GIVEN an api version vi and Kind ConfigMap
	// WHEN supported workloadtype is retrieved
	// THEN verify that the workloadType is empty because ConfigMap is not a supported type.
	apiVerKind = "v1.ConfigMap"
	workloadType = GetSupportedWorkloadType(apiVerKind)
	assert.Empty(workloadType)
}

// TestCreateServiceMonitorName test the creation of a service monitor name from relevant resources,
// as well as the creation of a legacy prometheus configmap job name
func TestCreateServiceMonitorName(t *testing.T) {
	tests := []struct {
		name                       string
		trait                      *vzapi.MetricsTrait
		portNum                    int
		expectedServiceMonitorName string
		expectedLegacyJobName      string
		expectError                bool
	}{
		{
			name:                       "test empty trait",
			trait:                      &vzapi.MetricsTrait{},
			portNum:                    0,
			expectedServiceMonitorName: "",
			expectError:                true,
		},
		{
			name: "test empty trait",
			trait: &vzapi.MetricsTrait{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
					Labels: map[string]string{
						appObjectMetaLabel:  "test-app",
						compObjectMetaLabel: "test-comp",
					},
				},
			},
			portNum:                    0,
			expectedServiceMonitorName: "test-app-test-namespace-test-comp",
			expectedLegacyJobName:      "test-app_test-namespace_test-comp",
			expectError:                false,
		},
		{
			name: "test name too long",
			trait: &vzapi.MetricsTrait{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
					Labels: map[string]string{
						appObjectMetaLabel:  "test-app-really-long-label",
						compObjectMetaLabel: "test-comp-extra-long-label",
					},
				},
			},
			portNum:                    1,
			expectedServiceMonitorName: "test-app-really-long-label-test-namespace-1",
			expectedLegacyJobName:      "test-app-really-long-label_test-namespace_1",
			expectError:                false,
		},
	}
	assert := asserts.New(t)
	for _, tt := range tests {
		smName, err1 := createServiceMonitorName(tt.trait, tt.portNum)
		jobName, err2 := createPrometheusScrapeConfigMapJobName(tt.trait, tt.portNum)
		if tt.expectError {
			assert.Error(err1)
			assert.Error(err2)
		} else {
			assert.NoError(err1)
			assert.NoError(err2)

		}
		asserts.Equal(t, tt.expectedServiceMonitorName, smName)
		asserts.Equal(t, tt.expectedLegacyJobName, jobName)
	}
}
