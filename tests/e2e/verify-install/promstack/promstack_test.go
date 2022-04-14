// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package promstack

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
	verrazzanoMonitoringNamespace   = "verrazzano-monitoring"
	waitTimeout                     = 3 * time.Minute
	pollingInterval                 = 10 * time.Second
	prometheusTLSSecret             = "prometheus-operator-kube-p-admission"
	prometheusOperatorDeployment    = "prometheus-operator-kube-p-operator"
	prometheusOperatorContainerName = "kube-prometheus-stack"
)

type enabledFunc func(string) bool

type enabledComponents struct {
	podName     string
	enabledFunc enabledFunc
}

var (
	promStackEnabledComponents = []enabledComponents{
		{podName: "prometheus-adapter", enabledFunc: pkg.IsPrometheusAdapterEnabled},
		{podName: "prometheus-operator-kube-p-operator", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{podName: "kube-state-metrics", enabledFunc: pkg.IsKubeStateMetricsEnabled},
		{podName: "prometheus-pushgateway", enabledFunc: pkg.IsPrometheusPushgatewayEnabled},
	}
	promOperatorCrds = []string{
		"alertmanagerconfigs.monitoring.coreos.com",
		"alertmanagers.monitoring.coreos.com",
		"podmonitors.monitoring.coreos.com",
		"probes.monitoring.coreos.com",
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"servicemonitors.monitoring.coreos.com",
		"thanosrulers.monitoring.coreos.com",
	}
	expectedPromOperatorArgs = []string{
		"--prometheus-default-base-image=ghcr.io/verrazzano/prometheus",
		"--alertmanager-default-base-image=ghcr.io/verrazzano/alertmanager",
	}
)

var t = framework.NewTestFramework("promstack")

// 'It' Wrapper to only run spec if the Prometheus Stack is supported on the current Verrazzano version
func WhenPromStackInstalledIt(description string, f func()) {
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
		t.Logs.Infof("Skipping check '%v', the Prometheus stack is not supported", description)
	}
}

var _ = t.Describe("Prometheus Stack", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenPromStackInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(verrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the pods are running
		// THEN we successfully find the running pods
		WhenPromStackInstalledIt("should have running pods", func() {
			promStackPodsRunning := func() bool {
				kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
				enabledPods := []string{}
				for _, component := range promStackEnabledComponents {
					if component.enabledFunc(kubeconfigPath) {
						enabledPods = append(enabledPods, component.podName)
					}
				}
				result, err := pkg.PodsRunning(verrazzanoMonitoringNamespace, enabledPods)
				if err != nil {
					AbortSuite(fmt.Sprintf("Pods %v is not running in the namespace: %v, error: %v", enabledPods, verrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the default images are from Verrazzano
		// THEN we see that the arguments are correctly populated
		WhenPromStackInstalledIt("should have the correct default images", func() {
			promStackPodsRunning := func() (bool, error) {
				return pkg.ContainerHasExpectedArgs(verrazzanoMonitoringNamespace, prometheusOperatorDeployment, prometheusOperatorContainerName, expectedPromOperatorArgs)
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPromStackInstalledIt("should have the correct Prometheus Operator CRDs", func() {
			verifyCRDList := func() (bool, error) {
				for _, crd := range promOperatorCrds {
					exists, err := pkg.DoesCRDExist(crd)
					if err != nil || !exists {
						return exists, err
					}
				}
				return true, nil
			}
			Eventually(verifyCRDList, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPromStackInstalledIt("should have the TLS secret", func() {
			Eventually(func() bool {
				return pkg.SecretsCreated(verrazzanoMonitoringNamespace, prometheusTLSSecret)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
