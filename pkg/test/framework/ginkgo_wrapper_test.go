// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"reflect"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

// TestVzAfterEach
func TestVzAfterEach(t *testing.T) {
	result := VzAfterEach(func() {})
	assert.True(t, result)
}

// TestVzAfterSuite
func TestVzAfterSuite(t *testing.T) {
	result := VzAfterSuite(func() {})
	assert.True(t, result)
}

// TestVzBeforeEach
func TestVzBeforeEach(t *testing.T) {
	result := VzBeforeEach(func() {})
	assert.True(t, result)
}

// TestVzBeforeSuite
func TestVzBeforeSuite(t *testing.T) {
	result := VzBeforeSuite(func() {})
	assert.True(t, result)
}

// TestVzContext
func TestVzContext(t *testing.T) {
	result := VzContext("Test Context", func() {})
	assert.True(t, result)
}

// TestVzCurrentGinkgoTestDescription
func TestVzCurrentGinkgoTestDescription(t *testing.T) {
	result := VzCurrentGinkgoTestDescription()
	assert.True(t, reflect.DeepEqual(result, ginkgo.CurrentSpecReport()))
}

// TestVzDescribe
func TestVzDescribe(t *testing.T) {
	result := VzDescribe("Test Describe", func() {})
	assert.True(t, result)
}

// TestVzIt
func TestVzIt(t *testing.T) {
	result := VzIt("Test It", func() {})
	assert.True(t, result)
}

// TestVzWhen
func TestVzWhen(t *testing.T) {
	result := VzWhen("Test When", func() {})
	assert.True(t, result)
}
