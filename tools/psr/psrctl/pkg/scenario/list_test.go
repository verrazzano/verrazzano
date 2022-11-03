// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAvailableScenarios tests the ListAvailableScenarios function
// GIVEN a directory with more than one scenario
//
//	WHEN the ListAvailableScenarios function is called
//	THEN ensure that the resulting scenario list is correct
func TestAvailableScenarios(t *testing.T) {
	sList, err := ListAvailableScenarios("./testdata")
	assert.NoError(t, err)
	assert.Equal(t, "OpenSearch-S1", sList[0].Name)
	assert.Equal(t, "ops-s1", sList[0].ID)
	assert.Equal(t, "ops-s1 description", sList[0].Description)

	assert.Equal(t, "OpenSearch-S2", sList[1].Name)
	assert.Equal(t, "ops-s2", sList[1].ID)
	assert.Equal(t, "ops-s2 description", sList[1].Description)
}
