// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"context"
	"fmt"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

var t = framework.NewTestFramework("verify-upgrade-required")

var waitTimeout = 3 * time.Minute
var pollingInterval = 10 * time.Second

var beforeSuite = t.BeforeSuiteFunc(func() {
	start := time.Now()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = BeforeSuite(beforeSuite)

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var afterSuite = t.AfterSuiteFunc(func() {
	start := time.Now()
	if failed || !beforeSuitePassed {
		dump.ExecuteBugReport()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Verify upgrade required when new version is available", Label("f:platform-lcm.upgrade", "f:observability.monitoring.prom"), func() {

	// This is a very specific check, which expects to run in the situation where we've updated the VPO to a
	// newer version but have not yet run an upgrade.  In that scenario the next CR edit must include an upgrade.
	// This is only valid for Release 1.3+, since before that release most post-install updates were not supported.

	// Verify that an edit to the system configuration is rejected when an upgrade is available but not yet applied
	// GIVEN a Verrazzano install
	// WHEN an edit is made without specifying an upgrade, but an upgrade to a newer version is available
	// THEN the edit is rejected by the webhook
	t.Context("Verify upgrade-required checks", func() {
		t.It("Upgrade-required validator test", func() {

			// Check the supported
			isSupportedVersion, err := minSupportedVersion()
			if err != nil {
				t.Fail(fmt.Sprintf("Error checking supported Verrazzano version: %s", err.Error()))
				return
			}
			if !isSupportedVersion {
				t.Logs.Infof("Test valid only for Verrazzano versions 1.3.0 and higher")
				return
			}

			var vz *vzalpha1.Verrazzano
			Eventually(func() error {
				vz, err = pkg.GetVerrazzano()
				return err
			}, waitTimeout, pollingInterval).Should(BeNil())
			if vz == nil {
				t.Fail(fmt.Sprintf("Unable to get Verrazzano instance"))
				return
			}

			if vz.Spec.Components.Istio == nil {
				vz.Spec.Components.Istio = &vzalpha1.IstioComponent{}
			}
			istio := vz.Spec.Components.Istio
			if istio.Ingress == nil {
				istio.Ingress = &vzalpha1.IstioIngressSection{
					Kubernetes: &vzalpha1.IstioKubernetesSection{},
				}
			}
			if istio.Egress == nil {
				istio.Egress = &vzalpha1.IstioEgressSection{
					Kubernetes: &vzalpha1.IstioKubernetesSection{},
				}
			}
			istio.Ingress.Kubernetes.Replicas = 3
			istio.Egress.Kubernetes.Replicas = 3

			vzclient, err := pkg.GetVerrazzanoClientset()
			if err != nil {
				t.Fail(fmt.Sprintf("Error getting Verrazzano client: %s", err.Error()))
				return
			}

			// This should fail with a webhook validation error
			_, err = vzclient.VerrazzanoV1alpha1().Verrazzanos(vz.Namespace).Update(context.TODO(), vz, v1.UpdateOptions{})
			t.Logs.Infof("Returned error: %s", err.Error())
			Expect(err).Should(Not(BeNil()))
		})
	})
})

func minSupportedVersion() (bool, error) {
	bomData, err := k8sutil.GetInstalledBOMData("")
	if err != nil {
		return false, err
	}
	installedBOM, err := bom.NewBOMFromJSON(bomData)
	if err != nil {
		return false, err
	}
	vpoVersion, err := semver.NewSemVersion(installedBOM.GetVersion())
	if err != nil {
		return false, err
	}
	supportedVersion, err := semver.NewSemVersion("v1.3.0")
	if err != nil {
		return false, err
	}
	if vpoVersion.IsLessThan(supportedVersion) {
		t.Logs.Infof("Verrazzano is NOT at supported version for test: %s", vpoVersion.ToString())
		return false, nil
	}
	t.Logs.Infof("Verrazzano is at supported version (1.3.0+) for test: %s", vpoVersion.ToString())
	return true, nil
}
