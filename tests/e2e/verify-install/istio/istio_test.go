// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio_test

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var _ = ginkgo.Describe("Istio", func() {
	const istioNamespace = "istio-system"

	ginkgoExt.DescribeTable("namespace",
		func(name string) {
			gomega.Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(name)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
		ginkgoExt.Entry(fmt.Sprintf("%s namespace should exist", istioNamespace), istioNamespace),
	)

	ginkgoExt.DescribeTable("deployments",
		func(namespace string) {
			expectedDeployments := []string{
				"istio-egressgateway",
				"istio-ingressgateway",
				"istiod",
				"istiocoredns",
			}

			deploymentNames := func(deploymentList *appsv1.DeploymentList) []string {
				var deploymentNames []string
				for _, deployment := range deploymentList.Items {
					deploymentNames = append(deploymentNames, deployment.Name)
				}
				return deploymentNames
			}

			var deployments *appsv1.DeploymentList
			gomega.Eventually(func() (*appsv1.DeploymentList, error) {
				var err error
				deployments, err = pkg.ListDeployments(namespace)
				return deployments, err
			}, waitTimeout, pollingInterval).ShouldNot(gomega.BeNil())

			gomega.Expect(deployments).Should(gomega.WithTransform(deploymentNames, gomega.ContainElements(expectedDeployments)))
			gomega.Expect(len(deployments.Items)).To(gomega.Equal(len(expectedDeployments)))
		},
		ginkgoExt.Entry(fmt.Sprintf("%s namespace should contain expected list of deployments", istioNamespace), istioNamespace),
	)
})
