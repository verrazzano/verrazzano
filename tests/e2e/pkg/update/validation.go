// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	longWaitTimeout     = 10 * time.Minute
	longPollingInterval = 30 * time.Second
)

func ValidatePods(deployName string, labelName string, nameSpace string, expectedPodsRunning uint32, hasPending bool) {
	gomega.Eventually(func() error {
		var runningPods uint32
		var pendingPods = false

		err := wait.ExponentialBackoff(wait.Backoff{
			Duration: time.Second * 15,
			Factor:   1,
			Jitter:   0.2,
			Steps:    15,
		}, func() (bool, error) {
			var err error
			pods, err := pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: map[string]string{labelName: deployName}}, nameSpace)
			if err != nil {
				return false, err
			}
			runningPods, pendingPods = getReadyPods(pods)
			// Compare the number of running/pending pods to the expected numbers
			if runningPods != expectedPodsRunning || pendingPods != hasPending {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return err
		}
		if runningPods != expectedPodsRunning {
			return fmt.Errorf("Deployment name %s: expect %d running pods, but got %d", deployName, expectedPodsRunning, runningPods)
		}
		if pendingPods != hasPending {
			return fmt.Errorf("Deployment name %s: expect pending pods %t, but got %t", deployName, hasPending, pendingPods)
		}
		return nil
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeNil(), "expect to get correct number of running and pending pods")
}

// getReadyPods returns the number of pending pods in the provided pod list
// and a boolean value indicating if there are no pending pods.
func getReadyPods(pods []v12.Pod) (uint32, bool) {
	var runningPods uint32
	pendingPods := false
	for _, pod := range pods {
		pkg.Log(pkg.Info, "checking pod: "+pod.Name)
		var podReady = pod.ObjectMeta.DeletionTimestamp == nil
		// Count the pod as not ready if one of its containers is not running or not ready
		for _, container := range pod.Status.ContainerStatuses {
			pkg.Log(pkg.Info, "checking container: "+container.Name)
			pkg.Log(pkg.Info, fmt.Sprintf("container ready=%t, container running %s, container waiting %s", container.Ready, container.State.Running, container.State.Waiting))
			if !container.Ready || container.State.Running == nil {
				podReady = false
			}
		}
		pkg.Log(pkg.Info, fmt.Sprintf("pod status phase: %s, pod ready: %t", pod.Status.Phase, podReady))
		if pod.Status.Phase == v12.PodRunning && podReady {
			runningPods++
		}
		if pod.Status.Phase == v12.PodPending {
			pendingPods = true
		}
	}
	return runningPods, pendingPods
}

func ValidatePodMemoryRequest(labels map[string]string, nameSpace, containerPrefix string, expectedMemory string) {
	gomega.Eventually(func() bool {
		var err error
		pods, err := pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: labels}, nameSpace)
		if err != nil {
			return false
		}
		memoryMatchedContainers := 0
		for _, pod := range pods {
			for _, container := range pod.Spec.Containers {
				if !strings.HasPrefix(container.Name, containerPrefix) {
					continue
				}
				expectedNodeMemory, err := resource.ParseQuantity(expectedMemory)
				if err != nil {
					pkg.Log(pkg.Error, err.Error())
					return false
				}
				pkg.Log(pkg.Info,
					fmt.Sprintf("Checking container memory request %v to match the expected value %s",
						container.Resources.Requests.Memory(), expectedMemory))
				if *container.Resources.Requests.Memory() == expectedNodeMemory {
					memoryMatchedContainers++
				}
			}
		}
		return memoryMatchedContainers == len(pods)
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find container with right memory settings")
}
