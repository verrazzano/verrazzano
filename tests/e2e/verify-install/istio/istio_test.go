// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
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

	expectedDeployments := []string{
		"istio-egressgateway",
		"istio-ingressgateway",
		"istiod",
	}

	t.DescribeTable("deployments",
		func(namespace string) {

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

	t.DescribeTable("Verify expected replica counts",
		func(namespace string) {

			// Verify the correct number of pods for each deployment based on the profile & any overrides

			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Fail(err.Error())
			}

			vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
			if err != nil {
				Fail(err.Error())
			}

			deployments := []types.NamespacedName{
				{Name: "istio-egressgateway", Namespace: namespace},
				{Name: "istio-ingressgateway", Namespace: namespace},
				{Name: "istiod", Namespace: namespace},
			}
			expectedPods := map[string]uint32{
				deployments[0].String(): getEgressReplicaCount(vz),
				deployments[1].String(): getIngressReplicaCount(vz),
				deployments[2].String(): getPilotReplicaCount(vz),
			}
			Eventually(func() (map[string]uint32, error) {
				return pkg.CheckPodCounts(deployments, buildListOpts)
			}, waitTimeout, pollingInterval).Should(Equal(expectedPods))
		},
		t.Entry(fmt.Sprintf("%s namespace should contain expected pod counts", istioNamespace), istioNamespace),
	)
})

func buildListOpts(name types.NamespacedName) (metav1.ListOptions, error) {
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": name.Name,
		},
	}
	labelMap, err := metav1.LabelSelectorAsMap(&selector)
	if err != nil {
		return metav1.ListOptions{}, err
	}
	listOpts := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelMap).String()}
	return listOpts, nil
}

func getIngressReplicaCount(vz *vzapi.Verrazzano) uint32 {
	istio := vz.Spec.Components.Istio
	if istio != nil {
		if !isIstioEnabled(istio) {
			return 0
		}
		if istio.Ingress != nil {
			if istio.Ingress.Kubernetes != nil {
				return istio.Ingress.Kubernetes.Replicas
			}
		}
	}
	if pkg.IsProdProfile() {
		return 2
	}
	return 1
}

func getEgressReplicaCount(vz *vzapi.Verrazzano) uint32 {
	istio := vz.Spec.Components.Istio
	if istio != nil {
		if !isIstioEnabled(istio) {
			return 0
		}
		if istio.Egress != nil && istio.Egress.Kubernetes != nil {
			return istio.Egress.Kubernetes.Replicas
		}
	}
	if pkg.IsProdProfile() {
		return 2
	}
	return 1
}

func getPilotReplicaCount(vz *vzapi.Verrazzano) uint32 {
	istio := vz.Spec.Components.Istio
	if istio != nil && !isIstioEnabled(istio) {
		return 0
	}
	return 1
}

func isIstioEnabled(istio *vzapi.IstioComponent) bool {
	if istio.Enabled != nil {
		return *istio.Enabled
	}
	return true
}
