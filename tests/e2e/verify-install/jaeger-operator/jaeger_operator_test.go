// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaegeroperator

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	verrazzanoMonitoringNamespace = "verrazzano-monitoring"
	waitTimeout                   = 3 * time.Minute
	pollingInterval               = 10 * time.Second
	jaegerOperatorName            = "jaeger-operator"
	operatorImage                 = "ghcr.io/verrazzano/jaeger-operator"
)

var (
	jaegerOperatorCrds = []string{
		"jaegers.jaegertracing.io",
	}
	expectedJaegerImages = map[string]string{
		"JAEGER-AGENT-IMAGE":     "ghcr.io/verrazzano/jaeger-agent",
		"JAEGER-COLLECTOR-IMAGE": "ghcr.io/verrazzano/jaeger-collector",
		"JAEGER-QUERY-IMAGE":     "ghcr.io/verrazzano/jaeger-query",
		"JAEGER-INGESTER-IMAGE":  "ghcr.io/verrazzano/jaeger-ingester",
	}
)

var t = framework.NewTestFramework("jaegeroperator")

func isJaegerOperatorEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return pkg.IsJaegerOperatorEnabled(kubeconfigPath)
}

// 'It' Wrapper to only run spec if the Jaeger operator is supported on the current Verrazzano version
func WhenJaegerOperatorInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.3.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Jaeger Operator is not supported", description)
	}
}

var _ = t.Describe("Jaeger Operator", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenJaegerOperatorInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				if !isJaegerOperatorEnabled() {
					return true, nil
				}
				return pkg.DoesNamespaceExist(verrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the pods are running
		// THEN we successfully find the running pods
		WhenJaegerOperatorInstalledIt("should have running pods", func() {
			jaegerOperatorPodsRunning := func() bool {
				if !isJaegerOperatorEnabled() {
					return true
				}
				result, err := pkg.PodsRunning(verrazzanoMonitoringNamespace, []string{jaegerOperatorName})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", jaegerOperatorName, verrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(jaegerOperatorPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the default Jaeger images are from Verrazzano
		// THEN we see that the env is correctly populated
		WhenJaegerOperatorInstalledIt("should have the correct default Jaeger images", func() {
			verifyImages := func() bool {
				if isJaegerOperatorEnabled() {
					// Check if Jaeger operator is running with the expected Verrazzano Jaeger Operator image
					image, err := pkg.GetContainerImage(verrazzanoMonitoringNamespace, jaegerOperatorName, jaegerOperatorName)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Container %s is not running in the namespace: %s, error: %v", jaegerOperatorName, verrazzanoMonitoringNamespace, err))
						return false
					}
					if !strings.HasPrefix(image, operatorImage) {
						pkg.Log(pkg.Error, fmt.Sprintf("Container %s image %s is not running with the expected image %s in the namespace: %s", jaegerOperatorName, image, operatorImage, verrazzanoMonitoringNamespace))
						return false
					}
					// Check if Jaeger operator env has been set to use Verrazzano Jaeger images
					containerEnv, err := pkg.GetContainerEnv(verrazzanoMonitoringNamespace, jaegerOperatorName, jaegerOperatorName)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Not able to get the environment variables in the container %s, error: %v", jaegerOperatorName, err))
						return false
					}
					for name, val := range expectedJaegerImages {
						found := false
						for _, actualEnv := range containerEnv {
							if actualEnv.Name == name {
								if !strings.HasPrefix(actualEnv.Value, val) {
									pkg.Log(pkg.Error, fmt.Sprintf("The value %s of the env %s for the container %s does not have the image %s as expected",
										actualEnv.Value, actualEnv.Name, jaegerOperatorName, val))
									return false
								}
								found = true
							}
						}
						if !found {
							pkg.Log(pkg.Error, fmt.Sprintf("The env %s not set for the container %s", name, jaegerOperatorName))
							return false
						}
					}
				}
				return true
			}
			Eventually(verifyImages, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenJaegerOperatorInstalledIt("should have the correct Jaeger Operator CRDs", func() {
			verifyCRDList := func() (bool, error) {
				if isJaegerOperatorEnabled() {
					for _, crd := range jaegerOperatorCrds {
						exists, err := pkg.DoesCRDExist(crd)
						if err != nil || !exists {
							return exists, err
						}
					}
					return true, nil
				}
				return true, nil
			}
			Eventually(verifyCRDList, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
