// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/capi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"time"
)

const (
	waitTimeout       = 10 * time.Minute
	pollingInterval   = 10 * time.Second
	minimumK8sVersion = "1.22.0"
)

var (
	isCAPISupported     bool
	inClusterVZ         *v1beta1.Verrazzano
	isMinimumK8sVersion bool
	kubeconfig          = getKubeConfigOrAbort()
)

var t = framework.NewTestFramework("capi")

var _ = t.Describe("Cluster API ", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check to make sure the pods exist
		// THEN we successfully find the pods in the cluster
		WhenCapiInstalledIt("expected pods are running", func() {
			pods := []string{"capi-controller-manager", "capi-ocne-bootstrap-controller-manager", "capi-ocne-control-plane-controller-manager", "capoci-controller-manager"}
			Eventually(func() (bool, error) {
				result, err := pkg.PodsRunning(constants.VerrazzanoCAPINamespace, pods)
				if err != nil {
					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoCAPINamespace, err)
				}
				return result, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected CAPI Pods should be running")
		})
	})
})

// 'It' Wrapper to only run spec if the CAPI is supported on the current Verrazzano version and is installed
func WhenCapiInstalledIt(description string, f func()) {
	t.It(description, func() {
		var err error
		inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
		}
		isMinimumK8sVersion, err = k8sutil.IsMinimumk8sVersion(minimumK8sVersion)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to check Minimum k8s version: %v", err))
		}
		isCAPISupported, err = pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
		}
		isCAPIComponentStatusEnabled := vzcr.IsComponentStatusEnabled(inClusterVZ, capi.ComponentName)
		if isMinimumK8sVersion && isCAPISupported && !isCAPIComponentStatusEnabled {
			f()
		} else {
			t.Logs.Infof("Skipping check '%v', Cluster API  is not installed/supported on this cluster", description)
		}
	})
}

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}
