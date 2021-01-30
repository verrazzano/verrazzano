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
	istionetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioclientv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
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
				"grafana",
				"istio-citadel",
				"istio-egressgateway",
				"istio-galley",
				"istio-ingressgateway",
				"istio-pilot",
				"istio-policy",
				"istio-sidecar-injector",
				"istio-telemetry",
				"istiocoredns",
				"prometheus",
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

	const istioJob = "istio-init-crd-14-1.4.6"
	ginkgoExt.DescribeTable("job",
		func(namespace string, name string) {
			gomega.Expect(pkg.DoesJobExist(namespace, name)).To(gomega.BeTrue())
		},
		ginkgoExt.Entry(fmt.Sprintf("%s namespace should contain job %s", istioNamespace, istioJob), istioNamespace, istioJob),
	)

	ginkgoExt.DescribeTable("should be running with Mutual TLS enabled",
		func(namespace string) {
			ginkgo.By("Default mesh policy should have Mutual TLS enabled in permissive mode")
			istioClient := getIstioClientset()

			// TODO: Need to resolve which version of API to use in go.mod
			//			mp, err := istioClient.AuthenticationV1alpha1().MeshPolicies().Get("default", metav1.GetOptions{})
			//			gomega.Expect(err).Should(gomega.Not(gomega.HaveOccurred()))
			//			gomega.Expect(mp.Spec.Peers[0].GetMtls().Mode).To(gomega.Equal(istioauthv1alpha.MutualTls_PERMISSIVE))

			ginkgo.By("Multi-cluster destination rule configured for Mutual TLS")
			dr, err := istioClient.NetworkingV1alpha3().DestinationRules("istio-system").
				Get(context.TODO(), "istio-multicluster-destinationrule", metav1.GetOptions{})
			gomega.Expect(err).Should(gomega.Not(gomega.HaveOccurred()))
			gomega.Expect(dr.Spec.TrafficPolicy.GetTls().GetMode()).To(gomega.Equal(istionetworkingv1alpha3.ClientTLSSettings_ISTIO_MUTUAL))
		},
		ginkgoExt.Entry("check Mutual TLS enabled", istioNamespace),
	)

	ginkgoExt.DescribeTable("should have gateways configured",
		func(namespace string) {
			expectedGateways := []string{
				"istio-multicluster-egressgateway",
				"istio-multicluster-ingressgateway",
			}
			istioClient := getIstioClientset()

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

// getIstioClientset returns the clientset object for Istio
func getIstioClientset() *istioClient.Clientset {
	cs, err := istioClient.NewForConfig(pkg.GetKubeConfig())
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to get Istio clientset: %v", err))
	}
	return cs
}
