// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var t = framework.NewTestFramework("install")

var beforeSuite = t.BeforeSuiteFunc(func() {})
var _ = BeforeSuite(beforeSuite)
var afterSuite = t.AfterSuiteFunc(func() {})
var _ = AfterSuite(afterSuite)
var _ = t.AfterEach(func() {})


var _ = t.Describe("OCNE Cluster Driver", Label("TODO: appropriate label"), Serial, func() {
	t.Context("Cluster Creation", func() {
		t.It("creates an active cluster", func() {
			expected := 2
			result := 1 + 1
			Expect(result).To(Equal(expected))
		})
	})

	t.Context("Cluster Deletion", func() {
		expected := 0
		t.BeforeEach(func() {
			expected += 1
		})
		t.It("can delete an active cluster", func() {
			result := 1
			Expect(result).To(Equal(expected))
		})
		t.It("can delete ann incompletely provisioned cluster", func() {
			result := 2
			Expect(result).To(Equal(expected))
		})
	})
})