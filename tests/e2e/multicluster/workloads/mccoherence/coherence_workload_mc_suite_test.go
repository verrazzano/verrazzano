// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mccoherence

import (
	"flag"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

var skipDeploy bool
var skipUndeploy bool
var skipVerify bool

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to install the application")
	flag.BoolVar(&skipVerify, "skipVerify", false, "skipVerify skips the post deployment app validations")
}

func TestMultiClusterCoherenceApplication(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Test Suite to validate the support for VerrazzanoCoherenceWorkload in multi-cluster environment")
}
