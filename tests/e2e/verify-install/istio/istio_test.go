// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("istio")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Istio", Label("f:platform-lcm.install"), func() {
	const istioNamespace = "istio-system"

	t.DescribeTable("namespace",
		func(name string) {
			Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(name)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		t.Entry(fmt.Sprintf("%s namespace should exist", istioNamespace), istioNamespace),
	)

	t.DescribeTable("deployments",
		func(namespace string) {
			expectedDeployments := []string{
				"istio-egressgateway",
				"istio-ingressgateway",
				"istiod",
			}

			deploymentNames := func(deploymentList *appsv1.DeploymentList) []string {
				var deploymentNames []string
				for _, deployment := range deploymentList.Items {
					deploymentNames = append(deploymentNames, deployment.Name)
				}
				return deploymentNames
			}

			var deployments *appsv1.DeploymentList
			Eventually(func() (*appsv1.DeploymentList, error) {
				var err error
				deployments, err = pkg.ListDeployments(namespace)
				return deployments, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			Expect(deployments).Should(WithTransform(deploymentNames, ContainElements(expectedDeployments)))
		},
		t.Entry(fmt.Sprintf("%s namespace should contain expected list of deployments", istioNamespace), istioNamespace),
	)
})
