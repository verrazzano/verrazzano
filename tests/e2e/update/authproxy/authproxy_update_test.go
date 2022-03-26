// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"

	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	authProxyName   = "verrazzano-authproxy"
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var t = framework.NewTestFramework("update authproxy")
var kubeconfigPath string
var nodeCount uint32 = 1
var _ = t.BeforeSuite(func() {
	var err error
	kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}

	kindNodeCount := os.Getenv("KIND_NODE_COUNT")
	if len(kindNodeCount) > 0 {
		u, err := strconv.ParseUint(kindNodeCount, 10, 32)
		if err == nil {
			nodeCount = uint32(u)
		}
	}
})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Update authProxy", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-authproxy verify", Label("f:platform-lcm.authproxy-verify"), func() {
		t.It("authproxy default replicas", func() {
			waitForCRToBeReady()
			cr, err := pkg.GetVerrazzano()
			if err != nil {
				Fail(err.Error())
			}

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
			waitForCRToBeReady()
			cr, err := pkg.GetVerrazzano()
			if err != nil {
				Fail(err.Error())
			}
			if cr.Spec.Components.AuthProxy == nil {
				cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
			}
			if cr.Spec.Components.AuthProxy.Kubernetes == nil {
				cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
			}
			cr.Spec.Components.AuthProxy.Kubernetes.Replicas = nodeCount
			updateCR(cr)

			expectedRunning := nodeCount
			expectedPending := uint32(0)
			validatePods(authProxyName, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})

	t.Describe("verrazzano-authproxy update affinity", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity", func() {
			waitForCRToBeReady()
			cr, err := pkg.GetVerrazzano()
			if err != nil {
				Fail(err.Error())
			}
			if cr.Spec.Components.AuthProxy == nil {
				cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
			}
			if cr.Spec.Components.AuthProxy.Kubernetes == nil {
				cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
			}
			cr.Spec.Components.AuthProxy.Kubernetes.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
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
						},
					},
				},
			}
			updateCR(cr)

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

func waitForCRToBeReady() {
	// Wait for the Verrazzano CR to be Ready
	Eventually(func() error {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			return err
		}
		if cr.Status.State != vzapi.VzStateReady {
			return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(BeNil(), "Expected to get Verrazzano CR with Ready state")
}

func updateCR(cr *vzapi.Verrazzano) {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		Fail(err.Error())
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	if err != nil {
		Fail(fmt.Sprintf("error updating Verrazzano instance: %v", err))
	}
}
