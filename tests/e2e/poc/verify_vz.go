// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package poc

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/framework"
	"time"
)

var _ = ginkgo.Describe("Verify Verrazzano [Feature:Client]", func() {

	f := framework.NewDefaultFramework("client-test")

	ginkgo.It("Should check that the kubernetes client is reachable", func() {
		ginkgo.By("seeing if the verrazzano-system namespace is reachable.")
		framework.EventuallyBeTrue(func() (bool, error) {
			return doesNamespaceExist(f.ClientSet, "verrazzano-system")
		}, 3*time.Minute, 5*time.Second)
	})
})
