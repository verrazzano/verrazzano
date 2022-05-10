// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var skipDeploy bool
var skipUndeploy bool
var namespace string

// init initializes variables from command line arguments
func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to uninstall the application")
	flag.StringVar(&namespace, "namespace", generatedNamespace, "namespace is the app namespace")
}

// TestOAMWorkloads runs the OAM workload test suite that tests various OAM workload types
func TestOAMWorkloads(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Test Suite to validate the support for OAM workloads")
}
