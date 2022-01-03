// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"flag"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

var skipDeploy bool
var skipUndeploy bool

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to install the application")
}

func TestSockShopApplication(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Sock Shop Suite")
}
