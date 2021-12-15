// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"go.uber.org/zap"
	"reflect"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

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

//ItM wraps It to emit a metric
func ItM(log *zap.SugaredLogger, text string, body interface{}) bool {
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	return ginkgo.It(text, func() {
		metrics.Emit(log) // Starting point metric
		reflect.ValueOf(body).Call([]reflect.Value{})
	})
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

//AfterEachM wraps after each to emit a metric
func AfterEachM(log *zap.SugaredLogger, body interface{}) bool {
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	return ginkgo.AfterEach(func() {
		metrics.Emit(log.With(metrics.Started)) // Starting point metric
		reflect.ValueOf(body).Call([]reflect.Value{})
	})
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
		pkg.Log(pkg.Debug, fmt.Sprintf("Describe block %q started - placeholder for making API call to emit test related metric(s)", VzCurrentGinkgoTestDescription().LeafNodeText))
		reflect.ValueOf(body).Call([]reflect.Value{})
		pkg.Log(pkg.Debug, fmt.Sprintf("Describe block %q ended - placeholder for making API call to emit test related metric(s)", VzCurrentGinkgoTestDescription().LeafNodeText))
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
