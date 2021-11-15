// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var emitMetricInitialized = false

// VzBeforeSuite - wrapper function for ginkgo BeforeSuite
func VzBeforeSuite(body func(), timeout ...float64) bool {
	pkg.Log(pkg.Debug, "VzBeforeSuite wrapper")
	ginkgo.BeforeSuite(func() {
		pkg.Log(pkg.Info, "Emit metric in BeforeSuite - started")
		body()
		pkg.Log(pkg.Info, "Emit metric in BeforeSuite - ended")
	}, timeout...)
	return true
}

// VzAfterSuite - wrapper function for ginkgo AfterSuite
func VzAfterSuite(body func(), timeout ...float64) bool {
	pkg.Log(pkg.Debug, "VzAfterSuite wrapper")
	ginkgo.AfterSuite(func() {
		pkg.Log(pkg.Info, "Emit metric in AfterSuite - started")
		body()
		pkg.Log(pkg.Info, "Emit metric in AfterSuite - ended")
	}, timeout...)
	return true
}

// VzIt - wrapper function for ginkgo It
func VzIt(text string, body interface{}, timeout ...float64) bool {
	pkg.Log(pkg.Debug, "VzIt wrapper")
	ginkgo.It(text, body, timeout...)
	return true
}

// VzBeforeEach - wrapper function for ginkgo BeforeEach
func VzBeforeEach(body interface{}, timeout ...float64) bool {
	pkg.Log(pkg.Debug, "VzBeforeEach wrapper")
	ginkgo.BeforeEach(body, timeout...)
	return true
}

// VzAfterEach - wrapper function for ginkgo AfterEach
func VzAfterEach(body interface{}, timeout ...float64) bool {
	pkg.Log(pkg.Debug, "VzAfterEach wrapper")
	ginkgo.AfterEach(body, timeout...)
	return true
}

// VzDescribe - wrapper function for ginkgo Describe
func VzDescribe(text string, body func()) bool {
	pkg.Log(pkg.Debug, "VzDescribe wrapper")
	if !emitMetricInitialized {
		ginkgo.JustBeforeEach(func() {
			pkg.Log(pkg.Info, fmt.Sprintf("Emit metric for for begin of It %s", ginkgo.CurrentGinkgoTestDescription().TestText))
		})
		ginkgo.JustAfterEach(func() {
			pkg.Log(pkg.Info, fmt.Sprintf("Emit metric for for end of It %s", ginkgo.CurrentGinkgoTestDescription().TestText))
		})
		emitMetricInitialized = true
	}
	ginkgo.Describe(text, body)
	return true
}

// VzContext - wrapper function for ginkgo Context
func VzContext(text string, body func()) bool {
	pkg.Log(pkg.Debug, "VzContext wrapper")
	ginkgo.Context(text, body)
	return true
}

// VzCurrentGinkgoTestDescription - wrapper function for ginkgo CurrentGinkgoTestDescription
func VzCurrentGinkgoTestDescription() ginkgo.GinkgoTestDescription {
	pkg.Log(pkg.Debug, "VzCurrentGinkgoTestDescription wrapper")
	return ginkgo.CurrentGinkgoTestDescription()
}

// VzWhen - wrapper function for ginkgo When
func VzWhen(text string, body func()) bool {
	pkg.Log(pkg.Debug, "VzWhen wrapper")
	ginkgo.When(text, body)
	return true
}
