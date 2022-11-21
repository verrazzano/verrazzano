// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingress

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var istioInjection string

func init() {
	flag.StringVar(&istioInjection, "istioInjection", "enabled", "istioInjection enables the injection of istio side cars")
}

// TestConsoleIngress tests an ingress trait setup for console access.
func TestConsoleIngress(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Console Ingress Test Suite")
}
