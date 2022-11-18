// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

// TestMetricsBindingPostUpgrade tests the Metrics Binding status after an upgrade
func TestMetricsBindingPostUpgrade(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Metrics Binding Post-Upgrade Test Suite")
}
