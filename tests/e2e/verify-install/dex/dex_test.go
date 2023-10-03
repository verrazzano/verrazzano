// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"time"
)

var (
	isDexSupported bool
	isDexInstalled bool
	inClusterVZ    *v1beta1.Verrazzano
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second

	pkceClientSecret = "verrazzano-pkce" //nolint:gosec //#gosec G101
)

var t = framework.NewTestFramework("dex")

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	kubeConfigPath := getKubeConfigOrAbort()
	isDexSupported, err = pkg.IsVerrazzanoMinVersion(constants.VerrazzanoVersion1_7_0, kubeConfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to check Verrazzano min version %s: %v", constants.VerrazzanoVersion1_7_0, err))
	}

	inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeConfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
	}
	isDexInstalled = vzcr.IsDexEnabled(inClusterVZ)
})

// AfterEach wraps Ginkgo AfterEach to emit a metric
var _ = t.AfterEach(func() {})

var _ = BeforeSuite(beforeSuite)

func getKubeConfigOrAbort() string {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeConfigPath
}

// 'It' Wrapper to only run spec if the Dex is supported on the current Verrazzano version and is installed
func WhenDexInstalledIt(description string, f func()) {
	t.It(description, func() {
		if isDexSupported && isDexInstalled {
			f()
		} else {
			t.Logs.Infof("Skipping check '%v', Dex is not installed on this cluster", description)
		}
	})
}

var _ = t.Describe("Dex", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Dex is installed
		// WHEN we check to make sure the pods exist
		// THEN we successfully find the pods in the cluster
		WhenDexInstalledIt("expected pod is running", func() {
			pods := []string{"dex"}
			t.Logs.Infof("Expected Dex pods: %v", pods)

			Eventually(func() (bool, error) {
				result, err := pkg.PodsRunning(constants.DexNamespace, pods)
				if err != nil {
					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoMonitoringNamespace, err)
				}
				return result, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Dex pod should be running")
		})

		// GIVEN the Dex is installed
		// WHEN we check to make sure the ingress is created
		// THEN we successfully find the ingresses in the cluster
		WhenDexInstalledIt("Dex ingress exists", func() {
			ing := []string{constants.DexIngress}
			t.Logs.Infof("Expected Dex ingresses %v", ing)
			Eventually(func() (bool, error) {
				return pkg.IngressesExist(inClusterVZ, constants.DexNamespace, ing)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Dex Ingress should exist")
		})

		// GIVEN the Dex is installed
		// WHEN we check to make sure the PKCE secret is created
		// THEN we successfully find the secret in the cluster
		WhenDexInstalledIt("PKCE secret exists", func() {
			Eventually(func() bool {
				result, err := pkg.DoesSecretExist(constants.DexNamespace, pkceClientSecret)
				if err != nil {
					AbortSuite(fmt.Sprintf("Secret %s does not exist in the namespace: %v, error: %v", pkceClientSecret, constants.DexNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Failed to find PKCE secret")
		})
	})
})
