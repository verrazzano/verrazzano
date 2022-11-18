// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package hotrod

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
)

var namespace string

func init() {
	flag.StringVar(&namespace, "namespace", generatedNamespace, "namespace is the app namespace")
}

func TestJaegerHotrodTracing(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Jaeger Hotrod Application Tracing Suite")
}
