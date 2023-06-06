// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"github.com/onsi/ginkgo/v2"
	"testing"
)

func TestTopology(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OpenSearch Operator Install Suite")
}
