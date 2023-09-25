// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifycrds

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var isManagedCluster bool

func init() {
	flag.BoolVar(&isManagedCluster, "isManagedCluster", false, "isManagedCluster indicates if it is a managed-cluster profile or not")
}

func TestVerifyCRDs(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Verify CRDs After Uninstall Suite")
}
