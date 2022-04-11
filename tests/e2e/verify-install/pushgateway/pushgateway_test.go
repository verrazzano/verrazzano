// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

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
	prometheusPushgatewayPod      = "prometheus-pushgateway"
	waitTimeout                   = 3 * time.Minute
	pollingInterval               = 10 * time.Second
)

var t = framework.NewTestFramework("prometheus-pushgateway")

// 'It' Wrapper to only run spec if the Prometheus Pushgateway is supported on the current Verrazzano installation
func WhenPrometheusPushgatewayInstalledIt(description string, f interface{}) {
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

var _ = t.Describe("Prometheus Pushgateway", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		WhenPrometheusPushgatewayInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(verrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPrometheusPushgatewayInstalledIt("should have a running pod", func() {
			prometheusOperatorPodsRunning := func() bool {
				result, err := pkg.PodsRunning(verrazzanoMonitoringNamespace, []string{prometheusPushgatewayPod})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", prometheusPushgatewayPod, verrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(prometheusOperatorPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
