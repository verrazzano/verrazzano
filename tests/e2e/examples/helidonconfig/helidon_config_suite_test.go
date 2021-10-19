// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonconfig

import (
	"flag"
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

var skipInstall string
var skipUninstall string

func init() {
	flag.StringVar(&skipInstall, "skipInstall", "false", "skipInstall skips the call to install the application")
	flag.StringVar(&skipUninstall, "skipUninstall", "false", "skipUninstall skips the call to install the application")
}

func TestHelidonExample(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("helidon-config-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Helidon Config Suite", []ginkgo.Reporter{junitReporter})
}
