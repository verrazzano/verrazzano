// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonpodannotation

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// TestHelidonDeploymentPodAnnotation tests a helidon deployment workload for Prometheus metric scraping with a pod annotation preventing metrics scraping
func TestHelidonDeploymentPodAnnotation(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Helidon Deployment Workload with Pod Prometheus Annotation Test Suite")
}
