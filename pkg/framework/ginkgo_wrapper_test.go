// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"reflect"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

// TestVzAfterEach
func TestVzAfterEach(t *testing.T) {
	result := VzAfterEach(func() { return })
	assert.True(t, result)
}

// TestVzAfterSuite
func TestVzAfterSuite(t *testing.T) {
	result := VzAfterSuite(func() { return })
	assert.True(t, result)
}

// TestVzBeforeEach
func TestVzBeforeEach(t *testing.T) {
	result := VzBeforeEach(func() { return })
	assert.True(t, result)
}

// TestVzBeforeSuite
func TestVzBeforeSuite(t *testing.T) {
	result := VzBeforeSuite(func() { return })
	assert.True(t, result)
}

// TestVzContext
func TestVzContext(t *testing.T) {
	result := VzContext("Test Context", func() { return })
	assert.True(t, result)
}

// TestVzCurrentGinkgoTestDescription
func TestVzCurrentGinkgoTestDescription(t *testing.T) {
	result := VzCurrentGinkgoTestDescription()
	assert.True(t, reflect.DeepEqual(result, ginkgo.GinkgoTestDescription{}))
}

// TestVzDescribe
func TestVzDescribe(t *testing.T) {
	result := VzDescribe("Test Describe", func() { return })
	assert.True(t, result)
}

// TestVzIt
func TestVzIt(t *testing.T) {
	result := VzIt("Test It", func() { return })
	assert.True(t, result)
}
