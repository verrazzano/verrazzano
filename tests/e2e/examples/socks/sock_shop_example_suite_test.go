// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"flag"
	"github.com/onsi/ginkgo/v2"

	"testing"
)

var skipDeploy bool
var skipUndeploy bool
var namespace string

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to install the application")
	flag.StringVar(&namespace, "namespace", generatedNamespace, "namespace is the app namespace")
}

func TestSockShopApplication(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Sock Shop Suite")
}
