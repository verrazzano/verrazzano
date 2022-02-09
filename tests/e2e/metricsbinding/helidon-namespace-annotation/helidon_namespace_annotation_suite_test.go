// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonnamespaceannotation

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// TestHelidonDeploymentWorkload tests a helidon deployment workload for Prometheus metric scraping with a Metrics Template override in the namespace annotation
func TestHelidonDeploymentNamespaceAnnotation(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Helidon Deployment Workload with Namespace Template Annotation Test Suite")
}
