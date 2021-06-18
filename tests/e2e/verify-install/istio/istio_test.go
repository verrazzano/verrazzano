// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio_test

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	istioclientv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Istio", func() {
	const istioNamespace = "istio-system"

	ginkgoExt.DescribeTable("namespace",
		func(name string) {
			gomega.Expect(pkg.DoesNamespaceExist(name)).To(gomega.BeTrue())
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
			deployments := pkg.ListDeployments(namespace)
			gomega.Expect(deployments).Should(
				gomega.SatisfyAll(
					gomega.Not(gomega.BeNil()),
					gomega.WithTransform(deploymentNames, gomega.ContainElements(expectedDeployments)),
				),
			)
			gomega.Expect(len(deployments.Items)).To(gomega.Equal(len(expectedDeployments)))
		},
		ginkgoExt.Entry(fmt.Sprintf("%s namespace should contain expected list of deployments", istioNamespace), istioNamespace),
	)

	ginkgoExt.DescribeTable("should have gateways configured",
		func(namespace string) {
			expectedGateways := []string{
				"istio-ingressgateway",
			}
			istioClient := pkg.GetIstioClientset()

			gatewayNames := func(gatewayList *istioclientv1alpha3.GatewayList) []string {
				gatewayNames := []string{}
				for _, gateway := range gatewayList.Items {
					gatewayNames = append(gatewayNames, gateway.Name)
				}
				return gatewayNames
			}
			gateways, err := istioClient.NetworkingV1alpha3().Gateways(namespace).List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()), fmt.Sprintf("Error fetching gateways in %s namespace", namespace))
			gomega.Expect(gateways).Should(
				gomega.SatisfyAll(
					gomega.Not(gomega.BeNil()),
					gomega.WithTransform(gatewayNames, gomega.ContainElements(expectedGateways)),
				),
			)
		},
		ginkgoExt.Entry("check gateways configured", istioNamespace),
	)
})
