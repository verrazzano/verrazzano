// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"go.uber.org/zap"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 30 * time.Second
	waitTimeout          = 20 * time.Minute
	pollingInterval      = 30 * time.Second
	clusterTemplate      = "templates/cluster-template-addons-new-vcn.yaml"
)

var kubeSystemPods = []string{"etcd", "kube-apiserver", "kube-controller-manager", "kube-proxy", "kube-scheduler"}

var beforeSuite = t.BeforeSuiteFunc(func() {
	start := time.Now()
	ensureCAPIVarsInitialized()
	capiPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	start := time.Now()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = AfterSuite(afterSuite)

var t = framework.NewTestFramework("cluster-api")

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenClusterAPIInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v',clusterAPI is not supported", description)
	}
}

// Run as part of BeforeSuite
func capiPrerequisites() {
	t.Logs.Infof("Start capi pre-requisites fotr cluster '%s'", ClusterName)

	t.Logs.Infof("Create CAPI configuration yaml for cluster '%s'", ClusterName)
	Eventually(func() error {
		return clusterTemplateGenerate(ClusterName, clusterTemplate, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

// ensurePodsRunning checks whether the pods are ready in a given namespace
func ensurePodsRunning(clusterName, namespace string, expectedPods []string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	result, err := pkg.PodsRunningInClusterWithClient(namespace, expectedPods, k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

var _ = t.Describe("CAPI e2e tests ,", Label("f:platform-verrazzano.capi-e2e-tests"), Serial, func() {

	t.Context(fmt.Sprintf("Create CAPI cluster '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Create CAPI cluster", func() {
			Eventually(func() error {
				return triggerCapiClusterCreation(ClusterTemplateGeneratedFilePath, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Monitor Cluster Creation", func() {
			Eventually(func() error {
				return monitorCapiClusterCreation(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Monitor Cluster Creation")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return ensureCapiAccess(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

		WhenClusterAPIInstalledIt("Check pods in kube-system CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsRunning(ClusterName, "kube-system", kubeSystemPods, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

	})

	t.Context(fmt.Sprintf("Delete CAPI cluster '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Delete CAPI cluster", func() {
			Eventually(func() error {
				return triggerCapiClusterDeletion(ClusterName, CapiDefaultNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Delete CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Monitor Cluster Deletion", func() {
			Eventually(func() error {
				return monitorCapiClusterDeletion(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Monitor Cluster Deletion")
		})
	})
})
