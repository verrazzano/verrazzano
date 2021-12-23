// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingress

import (
	"github.com/onsi/gomega"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

// TestConsoleIngress tests an ingress trait setup for console access.
func TestConsoleIngress(t *testing.T) {
	gomega.RegisterFailHandler(FailHandler)
	ginkgo.RunSpecs(t, "Console Ingress Test Suite")
}
