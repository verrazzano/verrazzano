// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = Describe("verify app restart", func() {
	const bobNamespace = "bobs-books"
	const helidonNamespace = "hello-helidon"
	const springNamespace = "springboot"
	const todoNamespace = "todo-list"

	ginkgoExt.DescribeTable("applications pods should not be restarted by upgrade",
		func(namespace string) {
			pods, err := pkg.GetRestartedPods(namespace)
			Expect(err).To(BeNil(), fmt.Sprintf("GetRestartPods failed with error %v", err))
			for podName, count := range pods {
				// For now just log the restart count.  There will be cases where pods are restarted
				// that has nothing to do with upgrade such as a liveness proble failure
				pkg.Log(pkg.Error, fmt.Sprintf("Pod %s in namespace %s was restarted %v times\n", podName, namespace, count))
			}
		},
		ginkgoExt.Entry(fmt.Sprintf("application pods in %s namespace should not have been restarted ", bobNamespace), bobNamespace),
		ginkgoExt.Entry(fmt.Sprintf("application pods in %s namespace should not have been restarted ", todoNamespace), todoNamespace),
		ginkgoExt.Entry(fmt.Sprintf("application pods in %s namespace should not have been restarted ", helidonNamespace), helidonNamespace),
		ginkgoExt.Entry(fmt.Sprintf("application pods in %s namespace should not have been restarted ", springNamespace), springNamespace),
	)
})
