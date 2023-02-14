// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifycrds

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var mySQLOperatorEnabled bool

func init() {
	flag.BoolVar(&mySQLOperatorEnabled, "mySQLOperatorEnabled", true, "mySQLOperatorEnabled describes whether the mySQLOperator component is enabled")
}

func TestVerifyCRDs(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Verify CRDs After Uninstall Suite")
}
