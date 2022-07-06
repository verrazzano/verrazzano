package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testDir = "../../pkg/analysis/test/files"

const testIP = "132.23.234.24"

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeALine(testIP), testIP)
	assert.Contains(t, SanitizeALine("test.me.test.me"), "test")
}
