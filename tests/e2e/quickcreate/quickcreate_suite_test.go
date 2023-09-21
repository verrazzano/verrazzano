// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package quickcreate

import (
	"flag"
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"testing"
)

var (
	t           = framework.NewTestFramework("quickcreate")
	clusterType string
)

func init() {
	flag.StringVar(&clusterType, "clusterType", Ocneoci, "The type of Quick Create cluster to test")
}

func TestQuickCreate(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "CAPI QuickCreate Suite")
}
