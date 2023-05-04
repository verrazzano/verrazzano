// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/capi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"sync"
	"time"
)

const (
	waitTimeout       = 10 * time.Minute
	pollingInterval   = 10 * time.Second
	capiLabelValue    = "controller-manager"
	capiLabelKey      = "control-plane"
	minimumK8sVersion = "1.22.0"
)

var (
	isCAPISupported     bool
	isCAPIEnabled       bool
	inClusterVZ         *v1beta1.Verrazzano
	isMinimumK8sVersion bool
	mutex               = &sync.Mutex{}
)

var t = framework.NewTestFramework("capi")

type CAPIEnabledModifier struct {
}

type CAPIEnabledModifierV1beta1 struct {
}

type CAPIDisabledModifier struct {
}

type CAPIDisabledModifierV1beta1 struct {
}

func (c CAPIDisabledModifier) ModifyCR(cr *v1alpha1.Verrazzano) {
	cr.Spec.Components.CAPI = &v1alpha1.CAPIComponent{}
	t.Logs.Debugf("CAPIDisabledModifier CR: %v", cr.Spec)
}

func (c CAPIDisabledModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	disabled := false
	cr.Spec.Components.CAPI.Enabled = &disabled
	t.Logs.Debugf("CAPIDisabledModifierV1beta1 CR: %v", cr.Spec)
}

func (c CAPIEnabledModifier) ModifyCR(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.CAPI == nil {
		cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	}
	enabled := true
	cr.Spec.Components.CAPI.Enabled = &enabled
	t.Logs.Debugf("CAPIEnabledModifier CR: %v", cr.Spec)
}

func (c CAPIEnabledModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.CAPI == nil {
		cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	}
	enabled := true
	cr.Spec.Components.CAPI.Enabled = &enabled
	t.Logs.Debugf("CAPIEnabledModifierV1beta1 CR: %v", cr.Spec)
}

var _ = t.AfterEach(func() {})

var _ = AfterSuite(func() {
	m := CAPIDisabledModifierV1beta1{}
	if isMinimumK8sVersion {
		if isCAPIEnabled {
			update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
			//update.ValidatePods(capiLabelValue, capiLabelKey, constants.VerrazzanoCAPINamespace, uint32(0), false)
		}
	}
})

var _ = BeforeSuite(func() {
	mutex.Lock()
	defer mutex.Unlock()
	m := CAPIEnabledModifierV1beta1{}
	var err error

	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get/parse kubernetes version: %s", err.Error()))
	}
	kubeconfigPath := getKubeConfigOrAbort()
	inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
	}

	isMinimumK8sVersion, err = k8sutil.IsMinimumk8sVersion(minimumK8sVersion)
	if isMinimumK8sVersion {
		isCAPIComponentStatusEnabled := vzcr.IsComponentStatusEnabled(inClusterVZ, capi.ComponentName)
		isCAPISupported, err = pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
		if err != nil {
			AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
		}
		if isCAPISupported && !isCAPIComponentStatusEnabled {
			update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
			inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
			if err != nil {
				AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
			}
			isCAPIEnabled = vzcr.IsCAPIEnabled(inClusterVZ)
		}
		if isCAPISupported && isCAPIEnabled {
			update.ValidatePods(capiLabelValue, capiLabelKey, constants.VerrazzanoCAPINamespace, uint32(4), false)
		}
	}
})

var _ = t.Describe("Cluster API ", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check to make sure the pods exist
		// THEN we successfully find the pods in the cluster
		t.Logs.Infof("Context: isMinimumK8sVersion", isMinimumK8sVersion)
		WhenCapiInstalledIt("expected pods are running", func() {
			pods := []string{"capi-controller-manager", "capi-ocne-bootstrap-controller-manager", "capi-ocne-control-plane-controller-manager", "capoci-controller-manager"}
			Eventually(func() (bool, error) {
				result, err := pkg.PodsRunning(constants.VerrazzanoCAPINamespace, pods)
				if err != nil {
					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoCAPINamespace, err)
				}
				return result, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected CAPI Pods should be running")
		})
	})
})

// 'It' Wrapper to only run spec if the CAPI is supported on the current Verrazzano version and is installed
func WhenCapiInstalledIt(description string, f func()) {
	t.It(description, func() {
		isCAPIEnabled = vzcr.IsCAPIEnabled(inClusterVZ)
		t.Logs.Infof("WhenCapiInstalledIt", isCAPIEnabled)
		if isMinimumK8sVersion && isCAPISupported && isCAPIEnabled {
			f()
		} else {
			t.Logs.Infof("Skipping chgeck '%v', Cluster API  is not installed/supported on this cluster", description)
		}
	})
}

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}
