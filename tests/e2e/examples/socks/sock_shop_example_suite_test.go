// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"flag"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"testing"
)

var skipInstall string
var skipUninstall string

func init() {
	flag.StringVar(&skipInstall, "skipInstall", "false", "skipInstall skips the call to install the application")
	flag.StringVar(&skipUninstall, "skipUninstall", "false", "skipUninstall skips the call to install the application")
}

func TestSockShopApplication(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("sock-shop-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Sock Shop Suite", []ginkgo.Reporter{junitReporter})
}
