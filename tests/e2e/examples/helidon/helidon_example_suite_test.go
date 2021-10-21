// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"flag"
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

var skipDeploy string
var skipUndeploy string

func init() {
	flag.StringVar(&skipDeploy, "skipDeploy", "false", "skipDeploy skips the call to install the application")
	flag.StringVar(&skipUndeploy, "skipUndeploy", "false", "skipUndeploy skips the call to install the application")
}

func TestHelidonExample(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("hello-helidon-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Hello Helidon Suite", []ginkgo.Reporter{junitReporter})
}
