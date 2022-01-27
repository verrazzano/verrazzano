// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"go.uber.org/zap"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

type TestFramework struct {
	Pkg     string
	Metrics *zap.SugaredLogger
	Logs    *zap.SugaredLogger
}

func new() *TestFramework {
	t := new(TestFramework)
	// use runtime to get the caller and then parse it to figure out the package name
	pc, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	pkg := funcName[:lastDot]
	// whew!
	t.Pkg = pkg
	t.Metrics, _ = metrics.NewLogger(pkg, metrics.MetricsIndex)
	t.Logs, _ = metrics.NewLogger(pkg, metrics.TestLogIndex)
	return t
}

func Metrics() *zap.SugaredLogger {
	t := new()
	return t.Metrics
}

// AfterEach wraps Ginkgo AfterEach to emit a metric
func AfterEach(args ...interface{}) bool {
	t := new()
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}

	body := args[0]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}

	f := func() {
		metrics.Emit(t.Metrics.With(metrics.Duration, metrics.DurationMillis()))
		reflect.ValueOf(body).Call([]reflect.Value{})
	}
	args[0] = f

	return ginkgo.AfterEach(args...)
}

// BeforeEach wraps Ginkgo BeforeEach to emit a metric
func BeforeEach(args ...interface{}) bool {
	// (unused) t := new()
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}

	body := args[0]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}

	f := func() {

		reflect.ValueOf(body).Call([]reflect.Value{})

	}
	args[0] = f

	return ginkgo.BeforeEach(args...)
}

// It wraps Ginkgo It to emit a metric
func It(text string, args ...interface{}) bool {
	t := new()
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[len(args)-1]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	f := func() {
		metrics.Emit(t.Metrics.With(metrics.Status, metrics.Started)) // Starting point metric
		reflect.ValueOf(body).Call([]reflect.Value{})
	}

	args[len(args)-1] = f
	return ginkgo.It(text, args...)
}

// Describe wraps Ginkgo Describe to emit a metric
func Describe(text string, args ...interface{}) bool {
	t := new()
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[len(args)-1]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	f := func() {
		metrics.Emit(t.Metrics.With(metrics.Status, metrics.Started))
		reflect.ValueOf(body).Call([]reflect.Value{})
		metrics.Emit(t.Metrics.With(metrics.Duration, metrics.DurationMillis()))
	}
	args[len(args)-1] = f
	return ginkgo.Describe(text, args...)
}

// DescribeTable - wrapper function for Ginkgo DescribeTable
func DescribeTable(text string, args ...interface{}) bool {
	t := new()
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[0]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	funcType := reflect.TypeOf(body)
	f := reflect.MakeFunc(funcType, func(args []reflect.Value) (results []reflect.Value) {
		metrics.Emit(t.Metrics.With(metrics.Status, metrics.Started))
		rv := reflect.ValueOf(body).Call(args)
		metrics.Emit(t.Metrics.With(metrics.Duration, metrics.DurationMillis()))
		return rv
	})
	args[0] = f.Interface()
	return ginkgo.DescribeTable(text, args...)
}

// BeforeSuite - wrapper function for Ginkgo BeforeSuite
func BeforeSuite(body func()) bool {
	t := new()
	if body == nil {
		ginkgo.Fail("Unsupported body type - expected non-nil")
	}

	f := func() {
		metrics.Emit(t.Metrics.With(metrics.Status, metrics.Started))
		reflect.ValueOf(body).Call([]reflect.Value{})
	}
	return ginkgo.BeforeSuite(f)
}

// AfterSuite - wrapper function for Ginkgo AfterSuite
func AfterSuite(body func()) bool {
	t := new()
	if body == nil {
		ginkgo.Fail("Unsupported body type - expected non-nil")
	}

	f := func() {
		metrics.Emit(t.Metrics.With(metrics.Duration, metrics.DurationMillis()))
		reflect.ValueOf(body).Call([]reflect.Value{})
	}
	return ginkgo.AfterSuite(f)
}

// Entry - wrapper function for Ginkgo Entry
func Entry(description interface{}, args ...interface{}) ginkgo.TableEntry {
	// (unused) t := new()
	return ginkgo.Entry(description, args...)
}

// Fail - wrapper function for Ginkgo Fail
func Fail(message string, callerSkip ...int) {
	// (unused) t := new()
	ginkgo.Fail(message, callerSkip...)
}

// Context - wrapper function for Ginkgo Context
func Context(text string, args ...interface{}) bool {
	// (unused) t := new()
	return Describe(text, args...)
}

// When - wrapper function for Ginkgo When
func When(text string, args ...interface{}) bool {
	// (unused) t := new()
	return ginkgo.When(text, args...)
}

// SynchronizedBeforeSuite - wrapper function for Ginkgo SynchronizedBeforeSuite
func SynchronizedBeforeSuite(process1Body func() []byte, allProcessBody func([]byte)) bool {
	// (unused) t := new()
	return ginkgo.SynchronizedBeforeSuite(process1Body, allProcessBody)
}

// SynchronizedAfterSuite - wrapper function for Ginkgo SynchronizedAfterSuite
func SynchronizedAfterSuite(allProcessBody func(), process1Body func()) bool {
	// (unused) t := new()
	return ginkgo.SynchronizedAfterSuite(allProcessBody, process1Body)
}

//	JustBeforeEach - wrapper function for Ginkgo JustBeforeEach
func JustBeforeEach(args ...interface{}) bool {
	// (unused) t := new()
	return ginkgo.JustBeforeEach(args...)
}

// JustAfterEach - wrapper function for Ginkgo JustAfterEach
func JustAfterEach(args ...interface{}) bool {
	// (unused) t := new()
	return ginkgo.JustAfterEach(args...)
}

//BeforeAll - wrapper function for Ginkgo BeforeAll
func BeforeAll(args ...interface{}) bool {
	// (unused) t := new()
	return ginkgo.BeforeAll(args...)
}

//AfterAll - wrapper function for Ginkgo AfterAll
func AfterAll(args ...interface{}) bool {
	// (unused) t := new()
	return ginkgo.AfterAll(args...)
}

// VzBeforeSuite - wrapper function for ginkgo BeforeSuite
func VzBeforeSuite(body interface{}) bool {
	pkg.Log(pkg.Debug, "VzBeforeSuite wrapper")
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	ginkgo.BeforeSuite(func() {
		pkg.Log(pkg.Info, "BeforeSuite started - placeholder for making API call to emit test related metric(s)")
		reflect.ValueOf(body).Call([]reflect.Value{})
		pkg.Log(pkg.Info, "BeforeSuite ended - placeholder for making API call to emit test related metric(s)")
	})
	return true
}

// VzAfterSuite - wrapper function for ginkgo AfterSuite
func VzAfterSuite(body interface{}) bool {
	pkg.Log(pkg.Debug, "VzAfterSuite wrapper")
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	ginkgo.AfterSuite(func() {
		pkg.Log(pkg.Info, "AfterSuite started - placeholder for making API call to emit test related metric(s)")
		reflect.ValueOf(body).Call([]reflect.Value{})
		pkg.Log(pkg.Info, "AfterSuite ended - placeholder for making API call to emit test related metric(s)")
	})
	return true
}

// VzIt - wrapper function for ginkgo It
func VzIt(text string, body interface{}) bool {
	pkg.Log(pkg.Debug, "VzIt wrapper")
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	ginkgo.It(text, func() {
		startTime := time.Now()

		pkg.Log(pkg.Info, fmt.Sprintf("It block %q started - placeholder for making API call to emit test related metric(s)", VzCurrentGinkgoTestDescription().LeafNodeText))
		reflect.ValueOf(body).Call([]reflect.Value{})
		pkg.Log(pkg.Info, fmt.Sprintf("It block %q ended - placeholder for making API call to emit test related metric(s)", VzCurrentGinkgoTestDescription().LeafNodeText))

		endTime := time.Now()
		durationMillis := float64(endTime.Sub(startTime) / time.Millisecond)
		// push the metrics
		if EmitGauge(text, "duration", durationMillis) != nil {
			return
		}
		if IncrementCounter(text, "number_of_runs") != nil {
			return
		}
	})
	return true
}

// VzBeforeEach - wrapper function for ginkgo BeforeEach
func VzBeforeEach(body interface{}) bool {
	pkg.Log(pkg.Debug, "VzBeforeEach wrapper")
	ginkgo.BeforeEach(body)
	return true
}

// VzAfterEach - wrapper function for ginkgo AfterEach
func VzAfterEach(body interface{}) bool {
	pkg.Log(pkg.Debug, "VzAfterEach wrapper")
	ginkgo.AfterEach(body)
	return true
}

// VzDescribe - wrapper function for ginkgo Describe
func VzDescribe(text string, body func()) bool {
	ginkgo.Describe(text, func() {
		startTime := time.Now()
		pkg.Log(pkg.Debug, fmt.Sprintf("Describe block %q started - placeholder for making API call to emit test related metric(s)", VzCurrentGinkgoTestDescription().LeafNodeText))
		reflect.ValueOf(body).Call([]reflect.Value{})
		pkg.Log(pkg.Info, fmt.Sprintf("Describe block %q ended - placeholder for making API call to emit test related metric(s)", ginkgo.CurrentSpecReport().LeafNodeText))

		endTime := time.Now()
		durationMillis := float64(endTime.Sub(startTime) / time.Millisecond)

		if EmitGauge(text, "duration", durationMillis) != nil {
			return
		}
		if IncrementCounter(text, "number_of_runs") != nil {
			return
		}
	})
	return true
}

// VzContext - wrapper function for ginkgo Context
func VzContext(text string, body func()) bool {
	pkg.Log(pkg.Debug, "VzContext wrapper")
	ginkgo.Context(text, body)
	return true
}

// VzCurrentGinkgoTestDescription - wrapper function for ginkgo CurrentGinkgoTestDescription
func VzCurrentGinkgoTestDescription() ginkgo.SpecReport {
	pkg.Log(pkg.Debug, "VzCurrentGinkgoTestDescription wrapper")
	return ginkgo.CurrentSpecReport()
}

// VzWhen - wrapper function for ginkgo When
func VzWhen(text string, body func()) bool {
	pkg.Log(pkg.Debug, "VzWhen wrapper")
	ginkgo.When(text, body)
	return true
}
