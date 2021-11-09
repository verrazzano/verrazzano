// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify_app

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var _ = Describe("Post-upgrade", func() {
	const bobNamespace = "bobs-books"
	const helidonNamespace = "hello-helidon"
	const springNamespace = "istio-system"
	const todoNamespace = "istio-system"

	ginkgoExt.DescribeTable("applications pods should not be restarted by upgrade",
		func(namespace string) {
			pods, err := pkg.GetRestartedPods(namespace)
			Expect(err).To(BeNil(), fmt.Sprintf("GetRestartPods failed with error %v", err))
			for _, pod := range pods {
				pkg.Log(pkg.Error, fmt.Sprintf("Pod %s in namespace %s was restarted\n", pod.Name, namespace))
			}
			Expect(len(pods)).To(BeZero(), fmt.Sprintf("%v in namespace %s were restarted", len(pods), namespace))
		},
		ginkgoExt.Entry(fmt.Sprintf("%s applicaiton pods in %s namespace should not have been restart ", bobNamespace), bobNamespace),
		ginkgoExt.Entry(fmt.Sprintf("%s applicaiton pods in %s namespace should not have been restart ", todoNamespace), todoNamespace),
		ginkgoExt.Entry(fmt.Sprintf("%s applicaiton pods in %s namespace should not have been restart ", helidonNamespace), helidonNamespace),
		ginkgoExt.Entry(fmt.Sprintf("%s applicaiton pods in %s namespace should not have been restart ", springNamespace), springNamespace),
	)
})
