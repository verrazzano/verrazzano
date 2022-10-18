// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgConstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout                  = 3 * time.Minute
	pollingInterval              = 10 * time.Second
	veleroRestoreHelperConfigMap = "restic-restore-action-config"
	resticHelperImage            = "velero-restic-restore-helper"
)

var (
	veleroCrds = []string{
		"backups.velero.io",
		"backupstoragelocations.velero.io",
		"deletebackuprequests.velero.io",
		"downloadrequests.velero.io",
		"podvolumebackups.velero.io",
		"podvolumerestores.velero.io",
		"resticrepositories.velero.io",
		"restores.velero.io",
		"schedules.velero.io",
		"serverstatusrequests.velero.io",
		"volumesnapshotlocations.velero.io",
	}

	imagePrefix = pkg.GetImagePrefix()
)

var t = framework.NewTestFramework("velero")

func isVeleroEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return pkg.IsVeleroEnabled(kubeconfigPath)
}

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenVeleroInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.4.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Velero is not supported", description)
	}
}

var _ = t.Describe("Velero", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Velero is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenVeleroInstalledIt("should have a velero namespace", func() {
			Eventually(func() (bool, error) {
				if !isVeleroEnabled() {
					return true, nil
				}
				return pkg.DoesNamespaceExist(constants.VeleroNameSpace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Velero is installed
		// WHEN we check to make sure the restore helper configmap is created in velero namespace
		// THEN we see that configmap data has the right image set
		WhenVeleroInstalledIt("should have restore configmap created with valid velero image", func() {
			verifyImages := func() bool {
				if isVeleroEnabled() {

					cfgMap, err := pkg.GetConfigMap(fmt.Sprintf("%s-%s", pkgConstants.Velero, veleroRestoreHelperConfigMap), constants.VeleroNameSpace)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Unable to retrieve configmap %s in the namespace: %s, error: %v", veleroRestoreHelperConfigMap, constants.VeleroNameSpace, err))
						return false
					}
					expectedResticHelperImage := fmt.Sprintf("%s/verrazzano/%s", imagePrefix, resticHelperImage)
					if !strings.HasPrefix(cfgMap.Data["image"], expectedResticHelperImage) {
						pkg.Log(pkg.Error, fmt.Sprintf("Configmap %s does not have the expected image %s in the namespace: %s. Image found = %s ", veleroRestoreHelperConfigMap, expectedResticHelperImage, constants.VeleroNameSpace, cfgMap.Data["image"]))
						return false
					}
				}
				return true
			}
			Eventually(verifyImages, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenVeleroInstalledIt("should have the correct velero CRDs", func() {
			verifyCRDList := func() (bool, error) {
				if isVeleroEnabled() {
					for _, crd := range veleroCrds {
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
