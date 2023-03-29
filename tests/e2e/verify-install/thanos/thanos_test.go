// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("thanos")

var (
	isThanosSupported bool
	isThanosInstalled bool
	inClusterVZ       *v1alpha1.Verrazzano
)

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	kubeconfigPath := getKubeConfigOrAbort()
	isThanosSupported, err = pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
	}

	inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
	}
	isThanosInstalled = vzcr.IsThanosEnabled(inClusterVZ)
})

var _ = BeforeSuite(beforeSuite)

// 'It' Wrapper to only run spec if the Thanos is supported on the current Verrazzano version and is installed
func WhenThanosInstalledIt(description string, f func()) {
	t.It(description, func() {
		if isThanosSupported && isThanosInstalled {
			f()
		} else {
			t.Logs.Infof("Skipping check '%v', Thanos is not installed on this cluster", description)
		}
	})

}

var _ = t.Describe("Thanos", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Thanos is installed
		// WHEN we check to make sure the pods exist
		// THEN we successfully find the pods in the cluster
		WhenThanosInstalledIt("query and query frontend pods are running", func() {
			Eventually(func() (bool, error) {
				pods := []string{"thanos-query", "thanos-query-frontend"}
				result, err := pkg.PodsRunning(constants.VerrazzanoMonitoringNamespace, pods)
				if err != nil {
					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoMonitoringNamespace, err)
				}
				return result, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Thanos Pods should be running")
		})

		// GIVEN the Thanos is installed
		// WHEN we check to make sure the ingresses have been created
		// THEN we successfully find the ingresses in the cluster
		WhenThanosInstalledIt("query store and query frontend ingresses exist", func() {
			Eventually(func() (bool, error) {
				ingresses := []string{"thanos-query-frontend", "thanos-grpc"}
				return pkg.IngressesExist(inClusterVZ, constants.VerrazzanoSystemNamespace, ingresses)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Thanos Ingresses should exist")
		})
	})
})
