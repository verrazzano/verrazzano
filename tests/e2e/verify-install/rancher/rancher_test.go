// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/client-go/kubernetes"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

var (
	client     *kubernetes.Clientset
	httpClient *retryablehttp.Client
	kialiErr   error
)

var t = framework.NewTestFramework("rancher")
var kubeconfig = getKubeConfigOrAbort()

var beforeSuite = t.BeforeSuiteFunc(func() {
	client, kialiErr = k8sutil.GetKubernetesClientset()
	Expect(kialiErr).ToNot(HaveOccurred())
	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
})

var _ = BeforeSuite(beforeSuite)

// 'It' Wrapper to only run spec if Rancher is supported on the current Verrazzano installation
func WhenRancherInstalledIt(description string, f func()) {
	t.It(description, func() {
		inClusterVZ, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
		}
		isRancherEnabled := vzcr.IsRancherEnabled(inClusterVZ)

		if isRancherEnabled {
			f()
		} else {
			t.Logs.Infof("Skipping test '%v', Rancher is not installed on this cluster", description)
		}
	})
}

var _ = t.AfterEach(func() {})

var _ = t.Describe("Rancher", Label("f:platform-lcm.install"), func() {

	t.Context("after successful installation", func() {

		WhenRancherInstalledIt("should have a running pod", func() {
			kialiPodsRunning := func() bool {
				result, err := pkg.PodsRunning(kiali.ComponentNamespace, []string{pkg.KialiName})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", pkg.KialiName, kiali.ComponentNamespace, err))
				}
				return result
			}
			Eventually(kialiPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}
