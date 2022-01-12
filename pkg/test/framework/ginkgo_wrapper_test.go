// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

// TestAfterEach
func TestAfterEach(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.AfterEach(func() {})
	assert.True(t, result)
}

// TestAfterSuite
func TestAfterSuite(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.AfterSuite(func() {})
	assert.True(t, result)
}

// TestBeforeEach
func TestBeforeEach(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.BeforeEach(func() {})
	assert.True(t, result)
}

// TestBeforeSuite
func TestBeforeSuite(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.BeforeSuite(func() {})
	assert.True(t, result)
}

// TestContext
func TestContext(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.Context("Test Context", func() {})
	assert.True(t, result)
}

// TestDescribe
func TestDescribe(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.Describe("Test Describe", func() {})
	assert.True(t, result)
}

// TestIt
func TestIt(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.It("Test It", func() {})
	assert.True(t, result)
}

// TestWhen
func TestWhen(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.When("Test When", func() {})
	assert.True(t, result)
}

// TestJustBeforeEach
func TestJustBeforeEach(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.JustBeforeEach(func() {})
	assert.True(t, result)
}

// TestJustAfterEach
func TestJustAfterEach(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.JustAfterEach(func() {})
	assert.True(t, result)
}

// TestBeforeAll
func TestBeforeAll(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.BeforeAll(func() {})
	assert.True(t, result)
}

// TestAfterAll
func TestAfterAll(t *testing.T) {
	var f = NewTestFramework("test")
	result := f.AfterAll(func() {})
	assert.True(t, result)
}

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
