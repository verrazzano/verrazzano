// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"time"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
	capiLabelValue  = "cluster.x-k8s.io/provider"
	capiLabelKey    = "cluster-api"
)

var (
	isCapiSupported bool
	isCapiInstalled bool
	inClusterVZ     *v1beta1.Verrazzano
)

type CapiEnabledModifier struct {
}

type CapiEnabledModifierV1beta1 struct {
}

type CapiDisabledModifier struct {
}

type CapiDisabledModifierV1beta1 struct {
}

func (c CapiDisabledModifier) ModifyCR(cr *v1alpha1.Verrazzano) {
	cr.Spec.Components.CAPI = &v1alpha1.CAPIComponent{}
	t.Logs.Debugf("CapiDisabledModifier CR: %v", cr.Spec)
}

func (c CapiDisabledModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	disabled := false
	cr.Spec.Components.CAPI.Enabled = &disabled
	t.Logs.Debugf("CapiDisabledModifierV1beta1 CR: %v", cr.Spec)
}

func (c CapiEnabledModifier) ModifyCR(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.CAPI == nil {
		cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	}
	enabled := true
	cr.Spec.Components.CAPI.Enabled = &enabled
	t.Logs.Debugf("CapiEnabledModifier CR: %v", cr.Spec)
}

func (c CapiEnabledModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.CAPI == nil {
		cr.Spec.Components.CAPI = &v1beta1.CAPIComponent{}
	}
	enabled := true
	cr.Spec.Components.CAPI.Enabled = &enabled
	t.Logs.Debugf("CapiEnabledModifierV1beta1 CR: %v", cr.Spec)
}

var t = framework.NewTestFramework("capi")

var _ = t.AfterEach(func() {})

var afterSuite = t.AfterSuiteFunc(func() {
	m := CapiDisabledModifierV1beta1{}
	var err error
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get Verrazzano from the cluster: %v", err))
	}

	if isCapiInstalled {
		update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(capiLabelValue, capiLabelKey, constants.VerrazzanoCapiNamespace, uint32(0), false)
	}
})

var beforeSuite = t.BeforeSuiteFunc(func() {
	m := CapiEnabledModifierV1beta1{}
	var err error
	kubeconfigPath := getKubeConfigOrAbort()

	isCapiSupported, err = pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %v", err))
	}
	if isCapiSupported {
		update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
		update.ValidatePods(capiLabelValue, capiLabelKey, constants.VerrazzanoCapiNamespace, uint32(4), false)
	}
	inClusterVZ, err = pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
	isCapiInstalled = vzcr.IsCAPIEnabled(inClusterVZ)
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Cluster API ", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check to make sure the pods exist
		// THEN we successfully find the pods in the cluster
		WhenCapiInstalledIt("expected pods are running", func() {
			pods := []string{"capi-controller-manager", "capi-ocne-bootstrap-controller-manager", "capi-ocne-control-plane-controller-manager", "capoci-controller-manager"}
			Eventually(func() (bool, error) {
				result, err := pkg.PodsRunning(constants.VerrazzanoCapiNamespace, pods)
				if err != nil {
					t.Logs.Errorf("Pods %v are not running in the namespace: %v, error: %v", pods, constants.VerrazzanoCapiNamespace, err)
				}
				return result, err
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Capi Pods should be running")
		})
	})
})

var _ = AfterSuite(afterSuite)

// 'It' Wrapper to only run spec if the Capi is supported on the current Verrazzano version and is installed
func WhenCapiInstalledIt(description string, f func()) {
	t.It(description, func() {
		if isCapiSupported && isCapiInstalled {
			f()
		} else {
			t.Logs.Infof("Skipping check '%v', Cluster Api  is not installed on this cluster", description)
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
