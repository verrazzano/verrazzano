// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout               = 3 * time.Minute
	pollingInterval           = 10 * time.Second
	rancherBackupOperatorName = "rancher-backup"
)

var (
	cattleCrds = []string{
		"backups.resources.cattle.io",
		"resourcesets.resources.cattle.io",
		"restores.resources.cattle.io",
	}
)

var t = framework.NewTestFramework("rancherbackup")

func isRancherBackupEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return pkg.IsRancherBackupEnabled(kubeconfigPath)
}

// 'It' Wrapper to only run spec if the Rancher Backup is supported on the current Verrazzano version
func WhenRancherBackupInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersionEventually("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.4.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Rancher Backup is not supported", description)
	}
}

var _ = t.Describe("Rancher Backup", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Rancher Backup is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenRancherBackupInstalledIt(fmt.Sprintf("should have a %s namespace", constants.RancherBackupNamesSpace), func() {
			Eventually(func() (bool, error) {
				if !isRancherBackupEnabled() {
					return true, nil
				}
				return pkg.DoesNamespaceExist(constants.RancherBackupNamesSpace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the pods are running
		// THEN we successfully find the running pods
		WhenRancherBackupInstalledIt("should have running pods", func() {
			rancherBackupPodsRunning := func() bool {
				if !isRancherBackupEnabled() {
					return true
				}
				result, err := pkg.PodsRunning(constants.RancherBackupNamesSpace, []string{rancherBackupOperatorName})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", rancherBackupOperatorName, constants.RancherBackupNamesSpace, err))
				}
				return result
			}
			Eventually(rancherBackupPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Rancher Backup is installed
		// WHEN we check to make sure cattle crds are created
		// THEN we see that correct set of rancher backup crds are set up
		WhenRancherBackupInstalledIt("should have the correct rancher backup CRDs", func() {
			verifyCRDList := func() (bool, error) {
				if isRancherBackupEnabled() {
					for _, crd := range cattleCrds {
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
