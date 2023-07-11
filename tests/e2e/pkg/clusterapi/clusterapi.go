// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusterapi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	minimumK8sVersion        = "1.24.0"
	minimumVerrazzanoVersion = "1.6.0"
)

// WhenClusterAPIInstalledIt - 'It' Wrapper to only run spec if the ClusterAPI is supported on the current Verrazzano version and is installed
func WhenClusterAPIInstalledIt(t *framework.TestFramework, description string, f func()) {
	t.It(description, func() {
		capiInstalled, err := isClusterAPIInstalled()
		if err != nil {
			ginkgo.AbortSuite(err.Error())
		}
		if capiInstalled {
			f()
		} else {
			t.Logs.Infof("Skipping test '%v', Cluster API  is not installed/supported on this cluster", description)
		}
	})
}

// isClusterAPIInstalled - determine if Cluster API is installed on the cluster
func isClusterAPIInstalled() (bool, error) {
	kubeConfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return false, fmt.Errorf("Failed to get default kubeconfig path: %s", err.Error())
	}
	inClusterVZ, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeConfig)
	if err != nil {
		return false, fmt.Errorf("Failed to get Verrazzano from the cluster: %v", err)
	}
	isClusterAPIEnabled := vzcr.IsClusterAPIEnabled(inClusterVZ)
	isMinimumK8sVersion, err := k8sutil.IsMinimumk8sVersion(minimumK8sVersion)
	if err != nil {
		return false, fmt.Errorf("Failed to check minimum Kubernetes version: %v", err)
	}
	isClusterAPISupported, err := pkg.IsVerrazzanoMinVersion(minimumVerrazzanoVersion, kubeConfig)
	if err != nil {
		return false, fmt.Errorf("Failed to check Verrazzano version %s: %v", minimumVerrazzanoVersion, err)
	}
	isComponentStatusEnabled := vzcr.IsComponentStatusEnabled(inClusterVZ, clusterapi.ComponentName)
	if isMinimumK8sVersion && isClusterAPISupported && (isClusterAPIEnabled && isComponentStatusEnabled) {
		return true, nil
	}
	return false, nil
}
