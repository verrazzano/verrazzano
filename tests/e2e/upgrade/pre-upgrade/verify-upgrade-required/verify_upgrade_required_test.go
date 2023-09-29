// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"context"
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/verrazzano"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
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

			var vz *vzalpha1.Verrazzano
			Eventually(func() (*vzalpha1.Verrazzano, error) {
				var err error
				vz, err = pkg.GetVerrazzano()
				return vz, err
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).
				ShouldNot(BeNil(), "Unable to get Verrazzano instance")

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

			config, err := k8sutil.GetKubeConfig()
			if err != nil {
				t.Fail(fmt.Sprintf("Error getting kubeconfig: %s", err.Error()))
				return
			}
			vzClient, err := pkg.GetV1Beta1ControllerRuntimeClient(config)
			if err != nil {
				t.Fail(fmt.Sprintf("Error getting Verrazzano client: %s", err.Error()))
				return
			}

			// This should fail with a webhook validation error
			err = verrazzano.UpdateV1Alpha1(context.TODO(), vzClient, vz)
			t.Logs.Infof("Returned error: %s", err.Error())
			Expect(err).Should(HaveOccurred())
		})
	})
})
