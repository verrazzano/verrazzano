// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package e2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

//var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
//	// setup some common setup for the Suite
//	return nil
//}, func(data []byte) {
//	// Stuff to run on all Ginkgo nodes
//})
//
//var _ = ginkgo.SynchronizedAfterSuite(func() {
//	//Common Cleanup Stuff
//}, func() {
//	//Registered Cleanup things for a specific Suite
//})

func RunE2ETests(t *testing.T) {
	//Do some logging setup, etc.

	gomega.RegisterFailHandler(ginkgo.Fail)
	// Disable skipped tests unless they are explicitly requested.

	ginkgo.RunSpecs(t, "Verrazzano e2e tests")
}
