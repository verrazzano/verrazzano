// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

func Test_formatISMPayload(t *testing.T) {
	age := "12d"

	var tests = []struct {
		name        string
		policy      vzapi.RetentionPolicy
		containsStr string
	}{
		{
			"Should format with default values",
			vzapi.RetentionPolicy{},
			defaultMinIndexAge,
		},
		{
			"Should format with custom values",
			vzapi.RetentionPolicy{
				MinAge: &age,
			},
			age,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := formatISMPayload(tt.policy, systemISMPayloadTemplate)
			assert.NoError(t, err)
			assert.Contains(t, payload, tt.containsStr)
		})
	}
}

// Test_fixupOpenSearchReplicaCount tests the fixupOpenSearchReplicaCount function.
func Test_fixupOpenSearchReplicaCount(t *testing.T) {
	assert := assert.New(t)

	// GIVEN an OpenSearch pod with a http port
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN a command should be executed to get the cluster health information
	//   AND a command should be executed to update the cluster index settings
	//   AND no error should be returned
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createOpenSearchPod(context.Client(), "http")
	execCommand = fakeExecCommand
	fakeExecScenarioNames = []string{"fixupOpenSearchReplicaCount/get", "fixupOpenSearchReplicaCount/put"} //nolint,ineffassign
	fakeExecScenarioIndex = 0                                                                              //nolint,ineffassign
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "Failed to fixup Elasticsearch index template")

	// GIVEN an OpenSearch pod with no http port
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN an error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createOpenSearchPod(context.Client(), "tcp")
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.Error(err, "Error should be returned if there is no http port for elasticsearch pods")

	// GIVEN a Verrazzano resource with version 1.1.0 in the status
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.ActualCR().Status.Version = "1.1.0"
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the source version is 1.1.0 or later")

	// GIVEN a Verrazzano resource with OpenSearch disabled
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{}
	fakeExecScenarioIndex = 0
	falseValue := false
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.EffectiveCR().Spec.Components.Elasticsearch.Enabled = &falseValue
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the elasticsearch is not enabled")
}
