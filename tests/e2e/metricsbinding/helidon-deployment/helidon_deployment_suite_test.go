// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deploymentworkload

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var namespace string

func init() {
	flag.StringVar(&namespace, "namespace", generatedNamespace, "namespace is the app namespace")
}

// TestHelidonDeploymentWorkload tests a helidon deployment workload for Prometheus metric scraping
func TestHelidonDeploymentWorkload(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Helidon Deployment Workload Test Suite")
}
