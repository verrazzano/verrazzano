// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ValidatePods(deployName string, labelName string, nameSpace string, expectedPodsRunning uint32, hasPending bool) {
	gomega.Eventually(func() error {
		var err error
		pods, err := pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: map[string]string{labelName: deployName}}, nameSpace)
		if err != nil {
			return err
		}
		// Compare the number of running/pending pods to the expected numbers
		var runningPods uint32 = 0
		var pendingPods = false
		for _, pod := range pods {
			if pod.Status.Phase == v12.PodRunning {
				runningPods++
			}
			if pod.Status.Phase == v12.PodPending {
				pendingPods = true
			}
		}
		if runningPods != expectedPodsRunning {
			return fmt.Errorf("expect %d running pods, but got %d", expectedPodsRunning, runningPods)
		}
		if pendingPods != hasPending {
			return fmt.Errorf("expect pending pods %t, but got %t", hasPending, pendingPods)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "expect to get correct number of running and pending pods")
}
