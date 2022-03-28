// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"os"
	"strconv"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	authProxyName   = "verrazzano-authproxy"
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

type AuthProxyReplicasModifier struct {
	replicas uint32
}

type AuthProxyPodPerNodeAffintyModifier struct {
}

type AuthProxyDefaultModifier struct {
}

func (u AuthProxyReplicasModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
	}
	cr.Spec.Components.AuthProxy.Kubernetes.Replicas = u.replicas
}

func (u AuthProxyPodPerNodeAffintyModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity = &corev1.Affinity{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	list := cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	list = append(list, corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: nil,
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "app",
					Operator: "In",
					Values: []string{
						authProxyName,
					},
				},
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	})
	cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = list
}

func (u AuthProxyDefaultModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
}

var t = framework.NewTestFramework("update authproxy")

var nodeCount uint32 = 1

var _ = t.BeforeSuite(func() {
	kindNodeCount := os.Getenv("KIND_NODE_COUNT")
	if len(kindNodeCount) > 0 {
		u, err := strconv.ParseUint(kindNodeCount, 10, 32)
		if err == nil {
			nodeCount = uint32(u)
		}
	}
})

var _ = t.AfterSuite(func() {
	m := AuthProxyDefaultModifier{}
	update.UpdateCR(m)
	cr := update.GetCR()

	expectedRunning := uint32(1)
	expectedPending := uint32(0)
	if cr.Spec.Profile == "production" || cr.Spec.Profile == "" {
		expectedRunning = 2
	}
	validatePods(authProxyName, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
})

var _ = t.Describe("Update authProxy", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-authproxy verify", Label("f:platform-lcm.authproxy-verify"), func() {
		t.It("authproxy default replicas", func() {
			cr := update.GetCR()

			expectedRunning := uint32(1)
			expectedPending := uint32(0)
			if cr.Spec.Profile == "production" || cr.Spec.Profile == "" {
				expectedRunning = 2
			}
			validatePods(authProxyName, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})

	t.Describe("verrazzano-authproxy update replicas", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas", func() {
			m := AuthProxyReplicasModifier{replicas: nodeCount}
			update.UpdateCR(m)

			expectedRunning := nodeCount
			expectedPending := uint32(0)
			validatePods(authProxyName, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})

	t.Describe("verrazzano-authproxy update affinity", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity", func() {
			m := AuthProxyPodPerNodeAffintyModifier{}
			update.UpdateCR(m)

			expectedRunning := nodeCount - 1
			expectedPending := uint32(2)
			if nodeCount == 1 {
				expectedRunning = nodeCount
				expectedPending = uint32(0)
			}
			validatePods(authProxyName, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})
})

func validatePods(deployName string, nameSpace string, expectedPodsRunning uint32, expectedPodsPending uint32) {
	Eventually(func() bool {
		var err error
		pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running/pending pods to the expected numbers
		var runningPods uint32 = 0
		var pendingPods uint32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
			if pod.Status.Phase == corev1.PodPending {
				pendingPods++
			}
		}
		return runningPods == expectedPodsRunning && pendingPods == expectedPodsPending
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get correct number of running and pending pods")
}
