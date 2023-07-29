// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
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
	isThanosSupported     bool
	isThanosInstalled     bool
	isStoreGatewayEnabled bool
	isCompactorEnabled    bool
	isRulerEnabled        bool
	inClusterVZ           *v1alpha1.Verrazzano
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

	if isThanosInstalled {
		// check if storegateway component is enabled
		isStoreGatewayEnabled, err = isThanosComponentEnabledInOverrides(inClusterVZ.Spec.Components.Thanos.InstallOverrides.ValueOverrides, "storegateway")
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to process VZ CR Thanos overrides for the Store Gateway: %v", err))
		}
		// check if compactor component is enabled
		isCompactorEnabled, err = isThanosComponentEnabledInOverrides(inClusterVZ.Spec.Components.Thanos.InstallOverrides.ValueOverrides, "compactor")
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to process VZ CR Thanos overrides for the Compactor: %v", err))
		}
		isRulerEnabled, err = isThanosComponentEnabledInOverrides(inClusterVZ.Spec.Components.Thanos.InstallOverrides.ValueOverrides, "ruler")
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to process VZ CR Thanos overrides for the Ruler: %v", err))
		}
	}
})

// isThanosComponentEnabledInOverrides returns true if the specified Thanos component is enabled in the VZ CR overrides
func isThanosComponentEnabledInOverrides(overrides []v1alpha1.Overrides, thanosCompName string) (bool, error) {
	for _, override := range inClusterVZ.Spec.Components.Thanos.InstallOverrides.ValueOverrides {
		if override.Values != nil {
			jsonString, err := gabs.ParseJSON(override.Values.Raw)
			if err != nil {
				return false, err
			}
			if container := jsonString.Path(fmt.Sprintf("%s.enabled", thanosCompName)); container != nil {
				if enabled, ok := container.Data().(bool); ok {
					return enabled, nil
				}
			}
		}
	}
	return false, nil
}

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
		WhenThanosInstalledIt("expected pods are running", func() {
			pods := []string{"thanos-query", "thanos-query-frontend"}
			if isStoreGatewayEnabled {
				pods = append(pods, "thanos-storegateway")
			}
			if isCompactorEnabled {
				pods = append(pods, "thanos-compactor")
			}
			if isRulerEnabled {
				pods = append(pods, "thanos-ruler")
			}
			t.Logs.Infof("Expected Thanos pods: %v", pods)

			Eventually(func() (bool, error) {
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
		WhenThanosInstalledIt("Thanos ingresses exist", func() {
			ingresses := []string{"thanos-query-frontend", "thanos-query-store"}
			if isRulerEnabled {
				ingresses = append(ingresses, "thanos-ruler")
			}

			t.Logs.Infof("Expected Thanos ingresses %v", ingresses)
			Eventually(func() (bool, error) {
				return pkg.IngressesExist(inClusterVZ, constants.VerrazzanoSystemNamespace, ingresses)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Thanos Ingresses should exist")
		})

		// GIVEN the Thanos is installed
		// WHEN the ingresses exist
		// THEN they should be accessible
		WhenThanosInstalledIt("Thanos ingresses should be accessible", func() {
			if pkg.IsManagedClusterProfile() {
				Skip("Skip verifying ingress accessibility for managed clusters")
			}
			httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
			creds := pkg.EventuallyGetSystemVMICredentials()
			urls := []*string{
				inClusterVZ.Status.VerrazzanoInstance.ThanosQueryURL,
			}
			if isRulerEnabled {
				urls = append(urls, inClusterVZ.Status.VerrazzanoInstance.ThanosRulerURL)
			}

			Eventually(func() bool {
				for _, url := range urls {
					if !pkg.AssertURLAccessibleAndAuthorized(httpClient, *url, creds) {
						return false
					}
				}
				return true
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// GIVEN the Thanos is installed
		// WHEN the ruler is enabled
		// THEN the rule data from Prometheus should be synced
		WhenThanosInstalledIt("Thanos ruler should contain rules populated from Prometheus", func() {
			if !isRulerEnabled {
				Skip("Skipping Rule verification because Ruler is not enabled")
			}

			kubeconfigPath := getKubeConfigOrAbort()
			Eventually(func() (interface{}, error) {
				return pkg.GetRulesFromThanosRuler(kubeconfigPath)
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).ShouldNot(BeNil())
		})
	})
})
