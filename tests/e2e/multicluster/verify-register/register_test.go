// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"time"

	"github.com/onsi/ginkgo"
)

var waitTimeout = 30 * time.Minute
var pollingInterval = 30 * time.Second

var _ = ginkgo.Describe("Multi Cluster Verify Register",
	func() {
		ginkgo.It("has the expected namespaces", func() {
		})
	})
