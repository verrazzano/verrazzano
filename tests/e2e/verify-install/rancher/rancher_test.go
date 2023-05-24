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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("rancher")

// 'It' Wrapper to only run spec if Rancher is supported on the current Verrazzano installation
func WhenRancherInstalledIt(description string, f func()) {
	t.It(description, func() {
		kubeconfig := getKubeConfigOrAbort()
		inClusterVZ, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
		}
		isRancherEnabled := vzcr.IsComponentStatusEnabled(inClusterVZ, rancher.ComponentName)

		supported, err := pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
		}

		if isRancherEnabled && supported {
			f()
		} else {
			t.Logs.Infof("Skipping test '%v', not supported for the configuration installed on this cluster", description)
		}
	})
}

var _ = t.AfterEach(func() {})

var _ = t.Describe("Rancher", Label("f:platform-lcm.install"), func() {

	t.Context("after successful installation", func() {
		var clientset dynamic.Interface

		// Get dynamic client
		Eventually(func() (dynamic.Interface, error) {
			kubePath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				return nil, err
			}
			clientset, err = pkg.GetDynamicClientInCluster(kubePath)
			return clientset, err
		}, waitTimeout, pollingInterval).ShouldNot(BeNil())

		WhenRancherInstalledIt("kontainerdrivers must be ready", func() {
			driversActive := func() bool {
				cattleDrivers, err := listKontainerDrivers(clientset)
				if err != nil {
					return false
				}

				allActive := true
				// The condition of each driver must be active
				for _, driver := range cattleDrivers.Items {
					status := driver.UnstructuredContent()["status"].(map[string]interface{})
					conditions := status["conditions"].([]interface{})
					driverActive := false
					for _, condition := range conditions {
						conditionData := condition.(map[string]interface{})
						if conditionData["type"].(string) == "Active" && conditionData["status"].(string) == "True" {
							driverActive = true
							break
						}
					}
					if !driverActive {
						t.Logs.Infof("Driver %s not Active", driver.GetName())
						allActive = false
					}
				}
				return allActive
			}
			Eventually(driversActive, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenRancherInstalledIt("expected kontainerdrivers must exist", func() {

			expectedDriversFound := func() bool {
				cattleDrivers, err := listKontainerDrivers(clientset)
				if err != nil {
					return false
				}

				foundCount := 0
				// The condition of each driver must be active
				for _, driver := range cattleDrivers.Items {
					switch driver.GetName() {
					case "amazonelasticcontainerservice":
						foundCount++
					case "azurekubernetesservice":
						foundCount++
					case "googlekubernetesengine":
						foundCount++
					case "ociocneengine":
						foundCount++
					case "oraclecontainerengine":
						foundCount++
					}
				}
				return foundCount == 5
			}
			Eventually(expectedDriversFound, waitTimeout, pollingInterval).Should(BeTrue())
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

func listKontainerDrivers(clientset dynamic.Interface) (*unstructured.UnstructuredList, error) {
	cattleDrivers, err := clientset.Resource(schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: "kontainerdrivers",
	}).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			t.Logs.Info("No kontainerdrivers found")
		} else {
			t.Logs.Errorf("Failed to list kontainerdrivers: %v", err)
		}
	}
	return cattleDrivers, err
}
