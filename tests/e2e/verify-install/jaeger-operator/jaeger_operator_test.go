// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaegeroperator

import (
	"fmt"
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
			verifyImages := func() (bool, error) {
				if isJaegerOperatorEnabled() {
					return pkg.ContainerHasExpectedEnv(verrazzanoMonitoringNamespace, jaegerOperatorName, jaegerOperatorName, expectedJaegerImages)
				}
				return true, nil
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
