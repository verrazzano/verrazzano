// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	waitTimeout             = 5 * time.Minute
	pollingInterval         = 10 * time.Second
	jaegerESIndexCleanerJob = "jaeger-operator-jaeger-es-index-cleaner"
)

var t = framework.NewTestFramework("jaeger_system_test")

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

var _ = t.Describe("Multi Cluster Jaeger Installation Validation", Label("f:platform-lcm.install"),
	func() {

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN only the Jaeger collector pods are created in the managed cluster.
		t.It("Jaeger Collector pods must be running in managed cluster", func() {
			labels := map[string]string{
				"app.kubernetes.io/component": "collector",
				"app.kubernetes.io/instance":  "jaeger-verrazzano-managed-cluster",
			}
			Eventually(func() bool {
				deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(managedKubeconfig, constants.VerrazzanoMonitoringNamespace, labels)
				if err != nil {
					return false
				}
				for _, deployment := range deployments.Items {
					isRunning, err := pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, managedKubeconfig)
					if err != nil {
						return false
					}
					return isRunning
				}
				return false
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN only one Jaeger collector deployment is created in the managed cluster.
		t.It("Atmost only one Jaeger Collector pods must be running in managed cluster", func() {
			labels := map[string]string{
				"app.kubernetes.io/component": "collector",
			}
			Eventually(func() bool {
				deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(managedKubeconfig, constants.VerrazzanoMonitoringNamespace, labels)
				if err != nil {
					return false
				}
				if len(deployments.Items) > 1 {
					pkg.Log(pkg.Error, "Managed cluster cannot have more than one Jaeger collectors")
					return false
				}
				return deployments.Items[0].Labels["app.kubernetes.io/instance"] == "jaeger-verrazzano-managed-cluster"
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN pods for the Jaeger query component do NOT get created in the managed cluster.
		t.It("Jaeger Query pods must NOT be running in managed cluster", func() {
			labels := map[string]string{
				"app.kubernetes.io/component": "query",
				"app.kubernetes.io/instance":  "jaeger-verrazzano-managed-cluster",
			}
			isRunning := false
			deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(managedKubeconfig, constants.VerrazzanoMonitoringNamespace, labels)
			if err != nil {
				Fail(err.Error())
			}
			for _, deployment := range deployments.Items {
				var err error
				isRunning, err = pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, managedKubeconfig)
				if err != nil {
					Fail(err.Error())
				}
			}
			Expect(isRunning).Should(BeFalse())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN cronjob for ES index cleaner is NOT created in the managed cluster.
		t.It("Jaeger ES index cleaner cronjob must NOT be created in managed cluster", func() {
			present, err := pkg.DoesCronJobExist(managedKubeconfig, constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
			if err != nil {
				Fail(err.Error())
			}
			Expect(present).Should(BeFalse())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN the Jaeger collector pods are created in the admin cluster.
		t.It("Jaeger Collector pods must be running in admin cluster", func() {
			labels := map[string]string{
				"app.kubernetes.io/component": "collector",
				"app.kubernetes.io/instance":  "jaeger-operator-jaeger",
			}
			Eventually(func() bool {
				deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(adminKubeconfig, constants.VerrazzanoMonitoringNamespace, labels)
				if err != nil {
					return false
				}
				for _, deployment := range deployments.Items {
					isRunning, err := pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, adminKubeconfig)
					if err != nil {
						return false
					}
					return isRunning
				}
				return false
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN the Jaeger collector pods are created in the admin cluster.
		t.It("Jaeger Query pods must be running in admin cluster", func() {

			labels := map[string]string{
				"app.kubernetes.io/component": "query",
				"app.kubernetes.io/instance":  "jaeger-operator-jaeger",
			}
			Eventually(func() bool {
				deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(adminKubeconfig, constants.VerrazzanoMonitoringNamespace, labels)
				if err != nil {
					return false
				}
				for _, deployment := range deployments.Items {
					isRunning, err := pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, adminKubeconfig)
					if err != nil {
						return false
					}
					return isRunning
				}
				return false
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN cronjob for ES index cleaner is created in the admin cluster.
		t.It("Jaeger ES index cleaner cronjob must be created in admin cluster", func() {
			Eventually(func() bool {
				present, err := pkg.DoesCronJobExist(adminKubeconfig, constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
				if err != nil {
					return false
				}
				return present
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})
	})
