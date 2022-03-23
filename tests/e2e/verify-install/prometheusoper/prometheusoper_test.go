// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheusoper

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
	prometheusOperatorPod         = "prometheus-operator-kube-p-operator"
	prometheusTLSSecret           = "prometheus-operator-kube-p-admission"
	waitTimeout                   = 3 * time.Minute
	pollingInterval               = 10 * time.Second
)

var t = framework.NewTestFramework("prometheusoper")

// 'It' Wrapper to only run spec if the Prometheus Operator is supported on the current Verrazzano installation
func WhenPrometheusOperatorInstalledIt(description string, f interface{}) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Prometheus Operator is not supported", description)
	}
}

var _ = t.AfterEach(func() {})

var _ = t.Describe("Prometheus Operator", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		WhenPrometheusOperatorInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(verrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPrometheusOperatorInstalledIt("should have a running pod", func() {
			prometheusOperatorPodsRunning := func() bool {
				result, err := pkg.PodsRunning(verrazzanoMonitoringNamespace, []string{prometheusOperatorPod})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", prometheusOperatorPod, verrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(prometheusOperatorPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPrometheusOperatorInstalledIt("should have the correct CRDs", func() {
			crds := []string{
				"alertmanagerconfigs.monitoring.coreos.com",
				"alertmanagers.monitoring.coreos.com",
				"podmonitors.monitoring.coreos.com",
				"probes.monitoring.coreos.com",
				"prometheuses.monitoring.coreos.com",
				"prometheusrules.monitoring.coreos.com",
				"servicemonitors.monitoring.coreos.com",
				"thanosrulers.monitoring.coreos.com",
			}
			verifyCRDList := func() (bool, error) {
				for _, crd := range crds {
					exists, err := pkg.DoesCRDExist(crd)
					if err != nil || !exists {
						return exists, err
					}
				}
				return true, nil
			}
			Eventually(verifyCRDList, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPrometheusOperatorInstalledIt("should have the TLS secret", func() {
			Eventually(func() bool {
				return pkg.SecretsCreated(verrazzanoMonitoringNamespace, prometheusTLSSecret)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
