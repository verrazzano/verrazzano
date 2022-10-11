// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var (
	expectedPodsOperator = []string{"verrazzano-application-operator"}
	expectedPodsOam      = []string{"oam-kubernetes-runtime"}
	waitTimeout          = 10 * time.Minute
	pollingInterval      = 30 * time.Second
)

const (
	verrazzanoSystemNS = "verrazzano-system"
)

var t = framework.NewTestFramework("oam")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Verify OAM Infra", Label("f:infra-lcm"), func() {
	t.Describe("verrazzano-application-operator pod", func() {
		t.It("must be running", func() {
			Eventually(func() (bool, error) {
				return applicationOperatorPodRunning()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Describe("oam-kubernetes-runtime pod", func() {
		t.It("is running", func() {
			Eventually(func() (bool, error) {
				return kubernetesRuntimePodRunning()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

// podsRunningInVerrazzanoSystem checks if all of the named pods are running in the verrazzano-system namespace.
// returns true iff all of the names pods are running.
// if at least one of the pods is not running it returns false.
func podsRunningInVerrazzanoSystem(podNames []string) (bool, error) {
	// if the list is empty, return true
	if len(podNames) == 0 {
		return true, nil
	}

	// otherwise check each pod name in the list
	for _, podName := range podNames {
		found, err := pkg.DoesPodExist(verrazzanoSystemNS, podName)
		if err != nil {
			return false, err
		}
		if found {
			// pod exists, nothing to do
		} else {
			// the pod does not exist, return a false
			return false, nil
		}
	}
	return true, nil
}

func kubernetesRuntimePodRunning() (bool, error) {
	return podsRunningInVerrazzanoSystem(expectedPodsOam)
}

func applicationOperatorPodRunning() (bool, error) {
	return podsRunningInVerrazzanoSystem(expectedPodsOperator)
}
