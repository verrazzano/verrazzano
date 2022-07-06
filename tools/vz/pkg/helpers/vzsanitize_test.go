package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testIP = "127.255.255.255"
)

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeALine(testIP), testIP)
	assert.Contains(t, SanitizeALine("test.me.test.me"), "test")
}
