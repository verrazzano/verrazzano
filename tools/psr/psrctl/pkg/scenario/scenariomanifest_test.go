// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"testing"
)

// TestAvailableScenarios tests the ListScenarioManifests function
// GIVEN a directory with more than one scenario
//
//	WHEN the ListScenarioManifests function is called
//	THEN ensure that the resulting scenario list is correct
func TestAvailableScenarios(t *testing.T) {
	m := Manager{
		Log: vzlog.DefaultLogger(),
		Manifest: embedded.PsrManifests{
			ScenarioAbsDir: "./testdata",
		},
		Namespace: "default",
	}
	sList, err := m.ListScenarioManifests()
	assert.NoError(t, err)
	assert.Equal(t, "OpenSearch-S1", sList[0].Name)
	assert.Equal(t, "ops-s1", sList[0].ID)
	assert.Equal(t, "ops-s1 description", sList[0].Description)

	assert.Equal(t, "OpenSearch-S2", sList[1].Name)
	assert.Equal(t, "ops-s2", sList[1].ID)
	assert.Equal(t, "ops-s2 description", sList[1].Description)
}
