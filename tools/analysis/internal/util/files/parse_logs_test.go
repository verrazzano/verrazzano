// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package files

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var filePath = "../../../test-cluster-dumps/coherence-workload/cluster-dump/verrazzano-install/verrazzano-platform-operator-7969dc8cd9-s4zg6/logs.txt"
var filteredJson []LogMessage

func TestFilterLogsByTypeGood(t *testing.T) {
	filteredJson = FilterLogsByLevelComponent("any", "any", filePath)
	assert.True(t, len(filteredJson) > 0 && len(filteredJson) <= fileLength)
}

func TestFilterLogsByTypeOnlyComponent(t *testing.T) {
	filteredJson = FilterLogsByLevelComponent("any", "istio", filePath)
	assert.True(t, len(filteredJson) > 0 && len(filteredJson) <= fileLength)
}

func TestFilterLogsByTypeOnlyLevel(t *testing.T) {
	filteredJson = FilterLogsByLevelComponent("info", "any", filePath)
	assert.True(t, len(filteredJson) > 0 && len(filteredJson) <= fileLength)
}

func TestFilterLogsByTypeBothLevelAndComponent(t *testing.T) {
	filteredJson = FilterLogsByLevelComponent("info", "istio", filePath)
	assert.True(t, len(filteredJson) > 0 && len(filteredJson) <= fileLength)
}

func TestFilterLogsByTypeInvalidLevelInvalidComponent(t *testing.T) {
	filteredJson = FilterLogsByLevelComponent("invalidLevel", "invalidComponenet", filePath)
	assert.True(t, len(filteredJson) == 0)
}
