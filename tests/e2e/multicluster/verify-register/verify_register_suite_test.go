// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var minimalVerification bool

func init() {
	flag.BoolVar(&minimalVerification, "minimalVerification", false, "minimalVerification to perform minimal verification")
}

func TestVerifyRegister(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Register Managed Cluster multi-cluster Suite")
}
