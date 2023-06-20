// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
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
	clusterTemplate      = "./cluster-template-addons.yaml"
)

var rancherPods = []string{"rancher"}

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

// checkPodsRunning checks whether the pods are ready in a given namespace
func checkPodsRunning(namespace string, expectedPods []string) bool {
	result, err := pkg.SpecificPodsRunning(namespace, "app=rancher")
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

// Run as part of BeforeSuite
func capiPrerequisites() {
	t.Logs.Info("Start capi pre-requisites")

	t.Logs.Infof("Create CAPI configuration yaml for cluster '%s'", ClusterName)
	Eventually(func() error {
		return clusterTemplateGenerate(ClusterName, clusterTemplate, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func checkFileNameSet() error {
	if ClusterTemplateGeneratedFilePath == "" {
		return fmt.Errorf("Filename not set for generated template")
	}
	t.Logs.Infof("+++ Generated File Path name = %s +++", ClusterTemplateGeneratedFilePath)
	return nil
}

var _ = t.Describe("CAPI e2e tests ,", Label("f:platform-verrazzano.capi-e2e-tests"), Serial, func() {

	t.Context("Verify CAPI generated file name is set ", func() {
		WhenClusterAPIInstalledIt("Start CAPI Cluster creation", func() {
			Eventually(func() error {
				return triggerCapiClusterCreation(ClusterTemplateGeneratedFilePath, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "CAPI generated file name is set")
		})
	})

	/*
		t.Context("Rancher backup", func() {
			WhenClusterAPIInstalledIt("Start rancher backup", func() {
				Eventually(func() error {
					return CreateRancherBackupObject()
				}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher backup CRD")
			})

			WhenClusterAPIInstalledIt("Check backup progress after rancher backup object was created", func() {
				Eventually(func() error {
					return common.TrackOperationProgress("rancher", common.BackupResource, common.BackupRancherName, common.VeleroNameSpace, t.Logs)
				}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher backup operation completed successfully")
			})

		})

		t.Context("Disaster simulation", func() {
			WhenClusterAPIInstalledIt("Delete all users that were created as part of pre-suite", func() {
				Eventually(func() bool {
					return DeleteRancherUsers(common.RancherURL)
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Delete rancher user")
			})
		})

		t.Context("Rancher restore", func() {
			WhenClusterAPIInstalledIt("Start restore after rancher backup is completed", func() {
				Eventually(func() error {
					return CreateRancherRestoreObject()
				}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher restore CRD")
			})
			WhenClusterAPIInstalledIt("Check rancher restore progress", func() {
				Eventually(func() error {
					return common.TrackOperationProgress("rancher", common.RestoreResource, common.RestoreRancherName, common.VeleroNameSpace, t.Logs)
				}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher restore operation completed successfully")
			})
		})

		t.Context("Rancher Data and Infra verification", func() {
			WhenClusterAPIInstalledIt("After restore is complete wait for rancher pods to come up", func() {
				Eventually(func() bool {
					return checkPodsRunning(constants.RancherSystemNamespace, rancherPods)
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if rancher infra is up")
			})
			WhenClusterAPIInstalledIt("Verify users are present rancher restore is complete", func() {
				Eventually(func() bool {
					return VerifyRancherUsers(common.RancherURL)
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if rancher user has been restored successfully")
			})
		})
	*/
})
