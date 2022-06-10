// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpol

import (
	"flag"
	"os"
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

func isUsingCalico() bool {
	usingCalico := os.Getenv("CREATE_CLUSTER_USE_CALICO")
	return usingCalico == "true"
}

func TestSecurityNetworkPolicies(t *testing.T) {
	if !isUsingCalico() {
		t.Skip("Calico not enabled, skipping test")
		return
	}
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Verrazzano Network Policy Suite")
}
