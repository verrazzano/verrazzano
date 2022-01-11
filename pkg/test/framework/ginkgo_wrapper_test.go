// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"github.com/onsi/ginkgo/v2"
	"go.uber.org/zap"
	"reflect"
	"testing"

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

func TestNewTestFramework(t *testing.T) {
	type args struct {
		pkg string
	}
	tests := []struct {
		name string
		args args
		want *TestFramework
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewTestFramework(tt.args.pkg), "NewTestFramework(%v)", tt.args.pkg)
		})
	}
}

func TestTestFramework_AfterEach(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.AfterEach(tt.args.args...), "AfterEach(%v)", tt.args.args...)
		})
	}
}

func TestTestFramework_BeforeEach(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.BeforeEach(tt.args.args...), "BeforeEach(%v)", tt.args.args...)
		})
	}
}

func TestTestFramework_It(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		text string
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.It(tt.args.text, tt.args.args...), "It(%v, %v)", tt.args.text, tt.args.args...)
		})
	}
}

func TestTestFramework_Describe(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		text string
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.Describe(tt.args.text, tt.args.args...), "Describe(%v, %v)", tt.args.text, tt.args.args...)
		})
	}
}

func TestTestFramework_DescribeTable(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		text string
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.DescribeTable(tt.args.text, tt.args.args...), "DescribeTable(%v, %v)", tt.args.text, tt.args.args...)
		})
	}
}

func TestTestFramework_BeforeSuite(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		body func()
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.BeforeSuite(tt.args.body), "BeforeSuite(%v)", tt.args.body)
		})
	}
}

func TestTestFramework_AfterSuite(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		body func()
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.AfterSuite(tt.args.body), "AfterSuite(%v)", tt.args.body)
		})
	}
}

func TestTestFramework_Entry(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		description interface{}
		args        []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ginkgo.TableEntry
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.Entry(tt.args.description, tt.args.args...), "Entry(%v, %v)", tt.args.description, tt.args.args...)
		})
	}
}

func TestTestFramework_Fail(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		message    string
		callerSkip []int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			t.Fail(tt.args.message, tt.args.callerSkip...)
		})
	}
}

func TestTestFramework_Context(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		text string
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.Context(tt.args.text, tt.args.args...), "Context(%v, %v)", tt.args.text, tt.args.args...)
		})
	}
}

func TestTestFramework_When(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		text string
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.When(tt.args.text, tt.args.args...), "When(%v, %v)", tt.args.text, tt.args.args...)
		})
	}
}

func TestTestFramework_SynchronizedBeforeSuite(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		process1Body   func() []byte
		allProcessBody func([]byte)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.SynchronizedBeforeSuite(tt.args.process1Body, tt.args.allProcessBody), "SynchronizedBeforeSuite(%v, %v)", tt.args.process1Body, tt.args.allProcessBody)
		})
	}
}

func TestTestFramework_SynchronizedAfterSuite(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		allProcessBody func()
		process1Body   func()
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.SynchronizedAfterSuite(tt.args.allProcessBody, tt.args.process1Body), "SynchronizedAfterSuite(%v, %v)", tt.args.allProcessBody, tt.args.process1Body)
		})
	}
}

func TestTestFramework_JustBeforeEach(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.JustBeforeEach(tt.args.args...), "JustBeforeEach(%v)", tt.args.args...)
		})
	}
}

func TestTestFramework_JustAfterEach(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.JustAfterEach(tt.args.args...), "JustAfterEach(%v)", tt.args.args...)
		})
	}
}

func TestTestFramework_BeforeAll(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.BeforeAll(tt.args.args...), "BeforeAll(%v)", tt.args.args...)
		})
	}
}

func TestTestFramework_AfterAll(t1 *testing.T) {
	type fields struct {
		Pkg     string
		Metrics *zap.SugaredLogger
		Logs    *zap.SugaredLogger
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TestFramework{
				Pkg:     tt.fields.Pkg,
				Metrics: tt.fields.Metrics,
				Logs:    tt.fields.Logs,
			}
			assert.Equalf(t1, tt.want, t.AfterAll(tt.args.args...), "AfterAll(%v)", tt.args.args...)
		})
	}
}
