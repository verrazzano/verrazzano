// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

// TestMetricsBindingPreUpgrade tests the deployment of resources before upgrade to verify the Metrics Binding upgrade
func TestMetricsBindingPreUpgrade(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Metrics Binding Pre-Upgrade Test Suite")
}
