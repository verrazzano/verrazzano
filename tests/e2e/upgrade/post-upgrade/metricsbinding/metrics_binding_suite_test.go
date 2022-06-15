// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// TestMetricsBindingPostUpgrade tests the Metrics Binding status after an upgrade
func TestMetricsBindingPostUpgrade(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Metrics Binding Post-Upgrade Test Suite")
}
