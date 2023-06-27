// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

//
//import (
//	"context"
//	"fmt"
//	"time"
//
//	. "github.com/onsi/ginkgo/v2"
//	. "github.com/onsi/gomega"
//	"github.com/verrazzano/verrazzano/pkg/constants"
//	"github.com/verrazzano/verrazzano/pkg/k8sutil"
//	"github.com/verrazzano/verrazzano/pkg/vzcr"
//	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusterapi"
//	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
//	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
//	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
//	"k8s.io/apimachinery/pkg/api/errors"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"k8s.io/client-go/dynamic"
//)
//
//const (
//	waitTimeout       = 5 * time.Minute
//	pollingInterval   = 10 * time.Second
//	minimumK8sVersion = "1.24.0"
//)
//
//var t = framework.NewTestFramework("clusterapi")
//
//var _ = t.Describe("Cluster API ", Label("f:platform-lcm.install"), func() {
//	t.Context("after successful installation", func() {
//		// GIVEN the Cluster API is installed
//		// WHEN we check to make sure the pods exist
//		// THEN we successfully find the pods in the cluster
//		WhenClusterAPIInstalledIt("expected pods are running", func() {
//			pods := []string{"capi-controller-manager", "capi-ocne-bootstrap-controller-manager", "capi-ocne-control-plane-controller-manager", "capoci-controller-manager"}
//			Eventually(func() (bool, error) {
//				result, err := pkg.PodsRunning(constants.VerrazzanoCAPINamespace, pods)
//				if err != nil {
//					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoCAPINamespace, err)
//				}
//				return result, err
//			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected ClusterAPI Pods should be running")
//		})
//	})
//})
//
//// 'It' Wrapper to only run spec if the ClusterAPI is supported on the current Verrazzano version and is installed
//func WhenClusterAPIInstalledIt(description string, f func()) {
//	t.It(description, func() {
//		kubeconfig := getKubeConfigOrAbort()
//		inClusterVZ, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfig)
//		if err != nil {
//			AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
//		}
//		isClusterAPIEnabled := vzcr.IsClusterAPIEnabled(inClusterVZ)
//		isMinimumK8sVersion, err := k8sutil.IsMinimumk8sVersion(minimumK8sVersion)
//		if err != nil {
//			AbortSuite(fmt.Sprintf("Failed to check Minimum k8s version: %v", err))
//		}
//		isClusterAPISupported, err := pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfig)
//		if err != nil {
//			AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
//		}
//		isComponentStatusEnabled := vzcr.IsComponentStatusEnabled(inClusterVZ, clusterapi.ComponentName)
//		if isMinimumK8sVersion && isClusterAPISupported && (isClusterAPIEnabled && isComponentStatusEnabled) {
//			f()
//		} else {
//			t.Logs.Infof("Skipping test '%v', Cluster API  is not installed/supported on this cluster", description)
//		}
//	})
//}
//
//var _ = t.Describe("KontainerDriver status", Label("f:platform-lcm.install"), func() {
//
//	t.Context("after successful installation", func() {
//		kubeconfig := getKubeConfigOrAbort()
//		inClusterVZ, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfig)
//		if err != nil {
//			AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
//		}
//		rancherConfigured := vzcr.IsComponentStatusEnabled(inClusterVZ, rancher.ComponentName)
//
//		var clientset dynamic.Interface
//
//		// Get dynamic client
//		Eventually(func() (dynamic.Interface, error) {
//			kubePath, err := k8sutil.GetKubeConfigLocation()
//			if err != nil {
//				return nil, err
//			}
//			clientset, err = pkg.GetDynamicClientInCluster(kubePath)
//			return clientset, err
//		}, waitTimeout, pollingInterval).ShouldNot(BeNil())
//
//		WhenClusterAPIInstalledIt("kontainerdrivers must be ready", func() {
//			if !rancherConfigured {
//				Skip("Skipping test because Rancher is not configured")
//			}
//			driversActive := func() bool {
//				cattleDrivers, err := listKontainerDrivers(clientset)
//				if err != nil {
//					return false
//				}
//
//				allActive := true
//				// The condition of each driver must be active
//				for _, driver := range cattleDrivers.Items {
//					status := driver.UnstructuredContent()["status"].(map[string]interface{})
//					conditions := status["conditions"].([]interface{})
//					driverActive := false
//					for _, condition := range conditions {
//						conditionData := condition.(map[string]interface{})
//						if conditionData["type"].(string) == "Active" && conditionData["status"].(string) == "True" {
//							driverActive = true
//							break
//						}
//					}
//					if !driverActive {
//						t.Logs.Infof("Driver %s not Active", driver.GetName())
//						allActive = false
//					}
//				}
//				return allActive
//			}
//			Eventually(driversActive, waitTimeout, pollingInterval).Should(BeTrue())
//		})
//
//		WhenClusterAPIInstalledIt("expected kontainerdrivers must exist", func() {
//			if !rancherConfigured {
//				Skip("Skipping test because Rancher is not configured")
//			}
//			expectedDriversFound := func() bool {
//				cattleDrivers, err := listKontainerDrivers(clientset)
//				if err != nil {
//					return false
//				}
//
//				foundCount := 0
//				// The condition of each driver must be active
//				for _, driver := range cattleDrivers.Items {
//					switch driver.GetName() {
//					case "amazonelasticcontainerservice":
//						foundCount++
//					case "azurekubernetesservice":
//						foundCount++
//					case "googlekubernetesengine":
//						foundCount++
//					case "ociocneengine":
//						foundCount++
//					case "oraclecontainerengine":
//						foundCount++
//					}
//				}
//				return foundCount == 5
//			}
//			Eventually(expectedDriversFound, waitTimeout, pollingInterval).Should(BeTrue())
//		})
//	})
//})
//
//func getKubeConfigOrAbort() string {
//	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
//	if err != nil {
//		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
//	}
//	return kubeconfigPath
//}
//
//func listKontainerDrivers(clientset dynamic.Interface) (*unstructured.UnstructuredList, error) {
//	cattleDrivers, err := clientset.Resource(schema.GroupVersionResource{
//		Group:    "management.cattle.io",
//		Version:  "v3",
//		Resource: "kontainerdrivers",
//	}).List(context.TODO(), metav1.ListOptions{})
//
//	if err != nil {
//		if errors.IsNotFound(err) {
//			t.Logs.Info("No kontainerdrivers found")
//		} else {
//			t.Logs.Errorf("Failed to list kontainerdrivers: %v", err)
//		}
//	}
//	return cattleDrivers, err
//}
