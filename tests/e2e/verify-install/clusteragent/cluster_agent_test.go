// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout           = 3 * time.Minute
	pollingInterval       = 10 * time.Second
	clusterAgentPodPrefix = "verrazzano-cluster-agent"
)

var t = framework.NewTestFramework("clusteragent")

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}

func clusterAgentInstalled() bool {
	kubeconfigPath := getKubeConfigOrAbort()
	isMinVersion160, err := pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	clusterAgentEnabled := pkg.IsClusterAgentEnabled(kubeconfigPath)
	return isMinVersion160 && clusterAgentEnabled
}

// 'It' Wrapper to only run spec if the Cluster Agent is supported on the current Verrazzano version
func WhenClusterAgentInstalledIt(description string, f func()) {
	if clusterAgentInstalled() {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Cluster Agent is not supported", description)
	}
}

// 'DescribeTable' Wrapper to only run spec if the Cluster Agent is supported on the current Verrazzano version
func WhenClusterAgentInstalledDescribetable(description string, args ...interface{}) {
	if clusterAgentInstalled() {
		t.DescribeTable(description, args...)
	} else {
		t.Logs.Infof("Skipping check '%v', the Cluster Agent is not supported", description)
	}
}

var _ = t.Describe("Cluster Agent", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {

		// GIVEN the Cluster Agent is installed
		// WHEN we check to make sure the pods are running
		// THEN we successfully find the running pods
		WhenClusterAgentInstalledIt("should have running pods", func() {
			podsRunning := func() (bool, error) {
				result, err := pkg.PodsRunning(constants.VerrazzanoSystemNamespace, []string{clusterAgentPodPrefix})
				if err != nil {
					t.Logs.Errorf("Pod %v is not running in the namespace: %v, error: %v", clusterAgentPodPrefix, constants.VerrazzanoSystemNamespace, err)
				}
				return result, err
			}
			Eventually(podsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Cluster Agent is installed
		// WHEN the installation is successful
		// THEN we find the expected CRDs
		WhenClusterAgentInstalledDescribetable("CRD for",
			func(name string) {
				Eventually(func() (bool, error) {
					return pkg.DoesCRDExist(name)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			t.Entry("multiclusterapplicationconfigurations should exist in cluster", "multiclusterapplicationconfigurations.clusters.verrazzano.io"),
			t.Entry("multiclustercomponents should exist in cluster", "multiclustercomponents.clusters.verrazzano.io"),
			t.Entry("multiclusterconfigmaps should exist in cluster", "multiclusterconfigmaps.clusters.verrazzano.io"),
			t.Entry("multiclustersecrets should exist in cluster", "multiclustersecrets.clusters.verrazzano.io"),
			t.Entry("verrazzanoprojects should exist in cluster", "verrazzanoprojects.clusters.verrazzano.io"),
		)
	})
})
