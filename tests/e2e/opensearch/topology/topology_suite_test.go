// Copyright (C) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package topology

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

func TestTopology(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "OpenSearch Topology Suite")
}
