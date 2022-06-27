// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
	veleroName      = "velero"
	operatorImage   = "ghcr.io/verrazzano/velero/velero"
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
	expectedVeleroImages = map[string]string{
		"VELERO":                       "ghcr.io/verrazzano/velero",
		"VELERO-PLUGIN-FOR-AWS":        "ghcr.io/verrazzano/velero-plugin-for-aws",
		"VELERO-RESTIC-RESTORE-HELPER": "ghcr.io/verrazzano/velero-restic-restore-helper",
	}
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
	supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.3.0: %s", err.Error()))
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
		// WHEN we check to make sure the default velero images are from Verrazzano
		// THEN we see that the env is correctly populated
		WhenVeleroInstalledIt("should have the correct default velero images", func() {
			verifyImages := func() bool {
				if isVeleroEnabled() {
					// Check if velero is running with the expected Verrazzano velero image
					image, err := pkg.GetContainerImage(constants.VeleroNameSpace, veleroName, veleroName)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Container %s is not running in the namespace: %s, error: %v", veleroName, constants.VeleroNameSpace, err))
						return false
					}
					if !strings.HasPrefix(image, operatorImage) {
						pkg.Log(pkg.Error, fmt.Sprintf("Container %s image %s is not running with the expected image %s in the namespace: %s", veleroName, image, operatorImage, constants.VeleroNameSpace))
						return false
					}
					// Check if velero env has been set to use Verrazzano velero images
					containerEnv, err := pkg.GetContainerEnv(constants.VeleroNameSpace, veleroName, veleroName)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Not able to get the environment variables in the container %s, error: %v", veleroName, err))
						return false
					}
					for name, val := range expectedVeleroImages {
						found := false
						for _, actualEnv := range containerEnv {
							if actualEnv.Name == name {
								if !strings.HasPrefix(actualEnv.Value, val) {
									pkg.Log(pkg.Error, fmt.Sprintf("The value %s of the env %s for the container %s does not have the image %s as expected",
										actualEnv.Value, actualEnv.Name, veleroName, val))
									return false
								}
								found = true
							}
						}
						if !found {
							pkg.Log(pkg.Error, fmt.Sprintf("The env %s not set for the container %s", name, veleroName))
							return false
						}
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
