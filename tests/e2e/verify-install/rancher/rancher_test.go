// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

var (
	clientset dynamic.Interface
)

var t = framework.NewTestFramework("rancher")
var kubeconfig = getKubeConfigOrAbort()

var beforeSuite = t.BeforeSuiteFunc(func() {
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

		WhenRancherInstalledIt("kontainerdrivers should be ready", func() {
			// Get dynamic client
			Eventually(func() (dynamic.Interface, error) {
				kubePath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return nil, err
				}
				clientset, err = pkg.GetDynamicClientInCluster(kubePath)
				return clientset, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			driversActive := func() bool {
				cattleDrivers, err := clientset.Resource(schema.GroupVersionResource{
					Group:    "management.cattle.io",
					Version:  "v3",
					Resource: "kontainerdrivers",
				}).List(context.TODO(), metav1.ListOptions{})
			}
			Eventually(driversActive, waitTimeout, pollingInterval).Should(BeTrue())

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
