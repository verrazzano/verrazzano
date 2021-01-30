// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package hello_helidon

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.BeforeSuite(func() {
	// deploy the application here
})

var _ = ginkgo.Describe("Helidon Hello World Example Application", func() {
	ginkgo.Context("Application", func() {
		ginkgo.It("Namespace should exist", func() {
			gomega.Expect(pkg.DoesNamespaceExist("hello-helidon")).To(gomega.BeTrue())
		})
		ginkgo.It("Pod should exist", func() {
			gomega.Expect(pkg.DoesPodExist("hello-helidon", "helidon-hello-world")).To(gomega.BeTrue())
		})
		// and so on
	})

	ginkgo.Context("Monitoring", func() {
		ginkgo.It("Metrics should be present in Prometheus", func() {
			// check for that
		})
	})
	// and so on
})

var _ = ginkgo.AfterSuite(func() {
	// undeploy the application here
})