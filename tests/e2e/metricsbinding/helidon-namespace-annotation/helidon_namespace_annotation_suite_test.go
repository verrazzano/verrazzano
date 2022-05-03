// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonnamespaceannotation

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var namespace string
var istioInjection string

func init() {
	flag.StringVar(&namespace, "namespace", generatedNamespace, "namespace is the app namespace")
	flag.StringVar(&istioInjection, "istioInjection", "enabled", "istioInjection enables the injection of istio side cars")
}

// TestHelidonDeploymentWorkload tests a helidon deployment workload for Prometheus metric scraping with a Metrics Template override in the namespace annotation
func TestHelidonDeploymentNamespaceAnnotation(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Helidon Deployment Workload with Namespace Template Annotation Test Suite")
}
