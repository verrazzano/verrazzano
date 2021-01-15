// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"strings"
	"sync"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
)

// GetVerrazzanoPassword returns the password credential for the verrazzano secret
func GetVerrazzanoPassword() string {
	secret := GetSecret("verrazzano-system", "verrazzano")
	return string(secret.Data["password"])
}

// Concurrently executes the given assertions in parallel and waits for them all to complete
func Concurrently(assertions ...func()) {
	number := len(assertions)
	var wg sync.WaitGroup
	wg.Add(number)
	for _, assertion := range assertions {
		go assert(&wg, assertion)
	}
	wg.Wait()
}

func assert(wg *sync.WaitGroup, assertion func()) {
	defer wg.Done()
	defer ginkgo.GinkgoRecover()
	assertion()
}

//PodsRunning checks if all the pods identified by namePrefixes are ready and running
func PodsRunning(namespace string, namePrefixes []string) bool {
	pods := ListPods(namespace)
	missing := notRunning(pods.Items, namePrefixes...)
	if len(missing) > 0 {
		Log(Info, fmt.Sprintf("Pods %v were NOT running in %v", missing, namespace))
	}
	return len(missing) == 0
}

// notRunning finds the pods not running
func notRunning(pods []v1.Pod, podNames ...string) []string {
	var notRunning []string
	for _, name := range podNames {
		running := isPodRunning(pods, name)
		if !running {
			notRunning = append(notRunning, name)
		}
	}
	return notRunning
}

// isPodRunning checks if the pod(s) with the name-prefix does exist and is running
func isPodRunning(pods []v1.Pod, namePrefix string) bool {
	running := false
	for i := range pods {
		if strings.HasPrefix(pods[i].Name, namePrefix) {
			running = isReadyAndRunning(pods[i])
			if !running {
				status := "status:"
				if len(pods[i].Status.ContainerStatuses) > 0 {
					for _, cs := range pods[i].Status.ContainerStatuses {
						//if cs.State.Waiting.Reason is CrashLoopBackOff, no need to retry
						if cs.State.Waiting != nil {
							status = fmt.Sprintf("%v %v", status, cs.State.Waiting.Reason)
						}
						if cs.State.Terminated != nil {
							status = fmt.Sprintf("%v %v", status, cs.State.Terminated.Reason)
						}
						if cs.LastTerminationState.Terminated != nil {
							status = fmt.Sprintf("%v %v", status, cs.LastTerminationState.Terminated.Reason)
						}
					}
				}
				Log(Info, fmt.Sprintf("  Pod %v was NOT running: %v \n", pods[i].Name, status))
				return false
			}
		}
	}
	return running
}

// isReadyAndRunning checks if the pod is ready and running
func isReadyAndRunning(pod v1.Pod) bool {
	if pod.Status.Phase == v1.PodRunning {
		for _, c := range pod.Status.ContainerStatuses {
			if !c.Ready {
				Log(Info, fmt.Sprintf("Pod %v container %v ready: %v", pod.Name, c.Name, c.Ready))
				return false
			}
		}
		return true
	}
	if pod.Status.Reason == "Evicted" && len(pod.Status.ContainerStatuses) == 0 {
		Log(Info, fmt.Sprintf("  Pod %v was Evicted\n", pod.Name))
		return true //ignore this evicted pod
	}
	return false
}
