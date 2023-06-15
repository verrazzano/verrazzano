// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var minimalVerification bool
var skipLogging bool

func init() {
	flag.BoolVar(&minimalVerification, "minimalVerification", false, "minimalVerification to perform minimal verification")
	flag.BoolVar(&skipLogging, "skipLogging", false, "skip logging test for registration verification")
}

func TestVerifyRegister(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Register Managed Cluster multi-cluster Suite")
}
