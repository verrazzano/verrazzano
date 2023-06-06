// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package topology

import (
	. "github.com/onsi/gomega"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var (
	t = framework.NewTestFramework("topology")
)

var _ = t.AfterEach(func() {})

var _ = t.Describe("Configure OpenSearch Topology", func() {

	t.It("can scale the cluster", func() {
		err := pkg.InstallOrUpdateOpenSearchOperator(t.Logs, 5, 3, 1)
		Expect(err).ToNot(HaveOccurred())

		pkg.EventuallyPodsReady(t.Logs, 5, 3, 1)

		err = pkg.InstallOrUpdateOpenSearchOperator(t.Logs, 5, 5, 1)
		Expect(err).ToNot(HaveOccurred())

		pkg.EventuallyPodsReady(t.Logs, 5, 5, 1)

		err = pkg.InstallOrUpdateOpenSearchOperator(t.Logs, 3, 3, 2)
		Expect(err).ToNot(HaveOccurred())

		pkg.EventuallyPodsReady(t.Logs, 3, 3, 2)
	})
})
