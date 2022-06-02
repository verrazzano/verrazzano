package files

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var filePath = "../../../test-cluster-dumps/coherence-workload/cluster-dump/verrazzano-install/verrazzano-platform-operator-7969dc8cd9-s4zg6/logs.txt"
var filteredJson = ""

func TestFilterLogsByTypeGood(t *testing.T) {
	filteredJson = FilterLogsByType("any", "any", filePath)
	assert.NotNil(t, filteredJson)
	assert.True(t, filteredLength <= fileLength)
}

func TestFilterLogsByTypeOnlyComponent(t *testing.T) {
	filteredJson = FilterLogsByType("any", "istio", filePath)
	assert.NotNil(t, filteredJson)
	assert.True(t, filteredLength > 0)
	assert.True(t, filteredLength <= fileLength)
}

func TestFilterLogsByTypeOnlyLevel(t *testing.T) {
	filteredJson = FilterLogsByType("info", "any", filePath)
	assert.NotNil(t, filteredJson)
	assert.True(t, filteredLength > 0)
	assert.True(t, filteredLength <= fileLength)
}

func TestFilterLogsByTypeBothLevelAndComponent(t *testing.T) {
	filteredJson = FilterLogsByType("info", "istio", filePath)
	assert.NotNil(t, filteredJson)
	assert.True(t, filteredLength > 0)
	assert.True(t, filteredLength <= fileLength)
}

func TestFilterLogsByTypeInvalidLevelInvalidComponent(t *testing.T) {
	filteredJson = FilterLogsByType("invalidLevel", "invalidComponenet", filePath)
	assert.True(t, filteredLength == 0)
}
