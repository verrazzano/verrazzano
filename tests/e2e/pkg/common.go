// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
)

// UsernamePassword - Username and Password credentials
type UsernamePassword struct {
	Username string
	Password string
}

// GetVerrazzanoPassword returns the password credential for the verrazzano secret
func GetVerrazzanoPassword() string {
	secret, _ := GetSecret("verrazzano-system", "verrazzano")
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

// AssertURLAccessibleAndAuthorized - Assert that a URL is accessible using the provided credentials
func AssertURLAccessibleAndAuthorized(client *retryablehttp.Client, url string, credentials *UsernamePassword) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(credentials.Username, credentials.Password)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	Log(Info, fmt.Sprintf("AssertURLAccessibleAndAuthorized %v Response:%v Error:%v", url, resp.StatusCode, err))
	return resp.StatusCode == http.StatusOK
}

//PodsRunning checks if all the pods identified by namePrefixes are ready and running
func PodsRunning(namespace string, namePrefixes []string) bool {
	pods := ListPods(namespace)
	missing := notRunning(pods.Items, namePrefixes...)
	if len(missing) > 0 {
		Log(Info, fmt.Sprintf("Pods %v were NOT running in %v", missing, namespace))
		for _, pod := range pods.Items {
			if isReadyAndRunning(pod) {
				Log(Debug, fmt.Sprintf("Pod %s ready", pod.Name))
			} else {
				Log(Info, fmt.Sprintf("Pod %s NOT ready: %v", pod.Name, pod.Status.ContainerStatuses))
			}
		}
	}
	return len(missing) == 0
}

//PodsNotRunning waits for all the pods in namePrefixes to be terminated
func PodsNotRunning(namespace string, namePrefixes []string) bool {
	allPods := ListPods(namespace)
	terminatedPods := notRunning(allPods.Items, namePrefixes...)
	var i int = 0
	for len(terminatedPods) != len(namePrefixes) {
		Log(Info, fmt.Sprintf("Pods %v were TERMINATED in %v", terminatedPods, namespace))
		time.Sleep(15 * time.Second)
		pods := ListPods(namespace)
		terminatedPods = notRunning(pods.Items, namePrefixes...)
		i++
		if i > 10 {
			break
		}
	}
	if len(terminatedPods) != len(namePrefixes) {
		runningPods := areRunning(allPods.Items, namePrefixes...)
		Log(Info, fmt.Sprintf("Pods %v were RUNNING in %v", runningPods, namespace))
		return false
	}
	Log(Info, fmt.Sprintf("ALL pods %v were TERMINATED in %v", terminatedPods, namespace))
	return true
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

// areRunning finds the pods that are running
func areRunning(pods []v1.Pod, podNames ...string) []string {
	var runningPods []string
	for _, name := range podNames {
		running := isPodRunning(pods, name)
		if running {
			runningPods = append(runningPods, name)
		}
	}
	return runningPods
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
				Log(Info, fmt.Sprintf("Pod %v was NOT running: %v", pods[i].Name, status))
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
		Log(Info, fmt.Sprintf("Pod %v was Evicted", pod.Name))
		return true //ignore this evicted pod
	}
	return false
}

func GetRetryPolicy(url string) func(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil {
			status := resp.StatusCode
			if status == http.StatusNotFound {
				return true, nil
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
}

// findMetric parses a Prometheus response to find a specified metric value
func findMetric(metrics []interface{}, key, value string) bool {
	for _, metric := range metrics {
		if Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

// MetricsExist validates the availability of a specified metric
func MetricsExist(metricsName, key, value string) bool {
	metrics := JTq(QueryMetric(metricsName), "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, key, value)
	} else {
		return false
	}
}

// JTq queries JSON text with a JSON path
func JTq(jtext string, path ...string) interface{} {
	var j map[string]interface{}
	json.Unmarshal([]byte(jtext), &j)
	return Jq(j, path...)
}

// Jq queries JSON nodes with a JSON path
func Jq(node interface{}, path ...string) interface{} {
	for _, p := range path {
		node = node.(map[string]interface{})[p]
	}
	return node
}

// SliceContainsString checks if the input slice (an array of strings)
// contains an entry which matches the string s
func SliceContainsString(slice []string, s string) bool {
	for _, str := range slice {
		if str == s {
			return true
		}
	}
	return false
}

// GetRequiredEnvVarOrFail returns the values of the provided environment variable name or fails.
func GetRequiredEnvVarOrFail(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		ginkgo.Fail(fmt.Sprintf("Environment variable '%s' required.", name))
	}
	return value
}
