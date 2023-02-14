// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogicworkload

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var istioInjection string

func init() {
	flag.StringVar(&istioInjection, "istioInjection", "enabled", "istioInjection enables the injection of istio side cars")
}

// TestWebLogicLoggingTrait tests an ingress trait setup for console access.
func TestWebLogicLoggingTrait(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "WebLogic Logging Trait Test Suite")
}
