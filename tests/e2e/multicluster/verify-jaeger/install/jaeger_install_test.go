// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
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
	testSkipMsgFmt          = "Cluster name is '%s'. Skipping tests meant for managed clusters"
)

var t = framework.NewTestFramework("jaeger_mc_system_test")

var kubeconfigPath = os.Getenv("KUBECONFIG")
var clusterName = os.Getenv("CLUSTER_NAME")

var _ = t.Describe("Multi Cluster Jaeger Installation Validation", Label("f:platform-lcm.install"), func() {

	// GIVEN a multicluster setup with an admin and a manged cluster,
	// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
	// THEN only the Jaeger collector pods are created in the managed cluster.
	t.It("Jaeger Collector pods must be running in managed cluster", func() {
		skipIfAdminCluster()
		labels := map[string]string{
			"app.kubernetes.io/component": "collector",
			"app.kubernetes.io/instance":  "jaeger-verrazzano-managed-cluster",
		}
		Eventually(func() bool {
			deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, labels)
			if err != nil {
				return false
			}
			for _, deployment := range deployments.Items {
				isRunning, err := pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, kubeconfigPath)
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
		skipIfAdminCluster()
		labels := map[string]string{
			"app.kubernetes.io/component": "collector",
		}
		Eventually(func() bool {
			deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, labels)
			if err != nil {
				return false
			}
			if len(deployments.Items) == 1 {
				// check if the only available Jaeger collector is the one managed by the mcagent.
				return deployments.Items[0].Labels["app.kubernetes.io/instance"] == "jaeger-verrazzano-managed-cluster"
			}
			pkg.Log(pkg.Error, "Managed cluster cannot have zero or more than one Jaeger collectors")
			return false
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})

	// GIVEN a multicluster setup with an admin and a manged cluster
	// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
	// THEN pods for the Jaeger query component do NOT get created in the managed cluster.
	t.It("Jaeger Query pods must NOT be running in managed cluster", func() {
		skipIfAdminCluster()
		labels := map[string]string{
			"app.kubernetes.io/component": "query",
			"app.kubernetes.io/instance":  "jaeger-verrazzano-managed-cluster",
		}
		isRunning := false
		deployments, err := pkg.ListDeploymentsMatchingLabelsInCluster(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, labels)
		if err != nil {
			Fail(err.Error())
		}
		for _, deployment := range deployments.Items {
			var err error
			isRunning, err = pkg.PodsRunningInCluster(constants.VerrazzanoMonitoringNamespace, []string{deployment.Name}, kubeconfigPath)
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
		skipIfAdminCluster()
		present, err := pkg.DoesCronJobExist(kubeconfigPath, constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
		if err != nil {
			Fail(err.Error())
		}
		Expect(present).Should(BeFalse())
	})
})

func skipIfAdminCluster() {
	if clusterName == "admin" {
		Skip(fmt.Sprintf(testSkipMsgFmt, clusterName))
	}
}
