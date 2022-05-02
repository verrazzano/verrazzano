// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var minimalVerification bool

func init() {
	flag.BoolVar(&minimalVerification, "minimalVerification", false, "minimalVerification to perform minimal verification")
}

func TestVerifyRegister(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Register Managed Cluster multi-cluster Suite")
}
