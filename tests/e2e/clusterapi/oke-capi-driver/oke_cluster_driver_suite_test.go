// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package okecapidriver

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var runAllTests bool

// init initializes variables from command line arguments
func init() {
	flag.BoolVar(&runAllTests, "runAllTests", false, "runAllTests toggles whether to run all cluster creation scenarios")
}

func TestOKEClusterDriver(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OKE CAPI Cluster Driver Suite")
}
