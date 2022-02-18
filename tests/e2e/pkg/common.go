// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// NumRetries - maximum number of retries
	NumRetries = 7

	// RetryWaitMin - minimum retry wait
	RetryWaitMin = 1 * time.Second

	// RetryWaitMax - maximum retry wait
	RetryWaitMax = 30 * time.Second
)

// UsernamePassword - Username and Password credentials
type UsernamePassword struct {
	Username string
	Password string
}

// GetVerrazzanoPassword returns the password credential for the Verrazzano secret
func GetVerrazzanoPassword() (string, error) {
	secret, err := GetSecret("verrazzano-system", "verrazzano")
	if err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// GetVerrazzanoPasswordInCluster returns the password credential for the Verrazzano secret in the "verrazzano-system" namespace for the given cluster
func GetVerrazzanoPasswordInCluster(kubeconfigPath string) (string, error) {
	secret, err := GetSecretInCluster("verrazzano-system", "verrazzano", kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Verrazzano secret: %v", err))
		return "", err
	}
	return string(secret.Data["password"]), nil
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
	if err != nil {
		Log(Error, fmt.Sprintf("AssertURLAccessibleAndAuthorized: URL=%v, Unexpected error=%v", url, err))
		return false
	}
	req.SetBasicAuth(credentials.Username, credentials.Password)
	resp, err := client.Do(req)
	if err != nil {
		Log(Error, fmt.Sprintf("AssertURLAccessibleAndAuthorized: URL=%v, Unexpected error=%v", url, err))
		return false
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("AssertURLAccessibleAndAuthorized: URL=%v, Unexpected status code=%v", url, resp.StatusCode))
		return false
	}
	// HTTP Server headers should never be returned.
	for headerName, headerValues := range resp.Header {
		if strings.EqualFold(headerName, "Server") {
			Log(Error, fmt.Sprintf("AssertURLAccessibleAndAuthorized: URL=%v, Unexpected Server header=%v", url, headerValues))
			return false
		}
	}
	return true
}

// PodsRunning is identical to PodsRunningInCluster, except that it uses the cluster specified in the environment
func PodsRunning(namespace string, namePrefixes []string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	return PodsRunningInCluster(namespace, namePrefixes, kubeconfigPath)
}

// PodsRunning checks if all the pods identified by namePrefixes are ready and running in the given cluster
func PodsRunningInCluster(namespace string, namePrefixes []string, kubeconfigPath string) bool {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting clientset for cluster, error: %v", err))
		return false
	}
	pods, err := ListPodsInCluster(namespace, clientset)
	if err != nil {
		Log(Error, fmt.Sprintf("Error listing pods in cluster for namespace: %s, error: %v", namespace, err))
		return false
	}
	missing := notRunning(pods.Items, namePrefixes...)
	if len(missing) > 0 {
		Log(Info, fmt.Sprintf("Pods %v were NOT running in %v", missing, namespace))
		for _, pod := range pods.Items {
			if isReadyAndRunning(pod) {
				Log(Debug, fmt.Sprintf("Pod %s ready", pod.Name))
			} else {
				Log(Info, fmt.Sprintf("Pod %s NOT ready: %v", pod.Name, formatContainerStatuses(pod.Status.ContainerStatuses)))
			}
		}
	}
	return len(missing) == 0
}

func formatContainerStatuses(containerStatuses []v1.ContainerStatus) string {
	output := ""
	for _, cs := range containerStatuses {
		output += fmt.Sprintf("Container name:%s ready:%s. ", cs.Name, strconv.FormatBool(cs.Ready))
	}
	return output
}

// PodsNotRunning returns true if all pods in namePrefixes are not running
func PodsNotRunning(namespace string, namePrefixes []string) (bool, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting clientset, error: %v", err))
		return false, err
	}
	allPods, err := ListPodsInCluster(namespace, clientset)
	if err != nil {
		Log(Error, fmt.Sprintf("Error listing pods in cluster for namespace: %s, error: %v", namespace, err))
		return false, err
	}
	terminatedPods := notRunning(allPods.Items, namePrefixes...)
	if len(terminatedPods) != len(namePrefixes) {
		runningPods := areRunning(allPods.Items, namePrefixes...)
		Log(Info, fmt.Sprintf("Pods %v were RUNNING in %v", runningPods, namespace))
		return false, nil
	}
	Log(Info, fmt.Sprintf("ALL pods %v were TERMINATED in %v", terminatedPods, namespace))
	return true, nil
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
				} else {
					status = fmt.Sprintf("%v %v", status, "unknown")
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

// GetRetryPolicy returns the standard retry policy
func GetRetryPolicy() func(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			if v, ok := err.(*neturl.Error); ok {
				//DefaultRetryPolicy does not retry "x509: certificate signed by unknown authority" which may happen on wildcard DNS (e.g. nip.io) when starting
				if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
					return HasWildcardDNS(v.URL), v
				}
			}
		}
		if resp != nil {
			status := resp.StatusCode
			if status == http.StatusNotFound {
				return true, nil
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
}

// GetRequiredEnvVarOrFail returns the values of the provided environment variable name or fails.
func GetRequiredEnvVarOrFail(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		ginkgo.Fail(fmt.Sprintf("Environment variable '%s' required.", name))
	}
	return value
}

// SlicesContainSameStrings compares two slices and returns true if they contain the same strings in any order
func SlicesContainSameStrings(strings1, strings2 []string) bool {
	if len(strings1) != len(strings2) {
		return false
	}
	if len(strings1) == 0 {
		return true
	}
	// count how many times each string occurs in case there are duplicates
	m1 := map[string]int32{}
	for _, s := range strings1 {
		m1[s]++
	}
	m2 := map[string]int32{}
	for _, s := range strings2 {
		m2[s]++
	}
	return reflect.DeepEqual(m1, m2)
}

// PolicyRulesEqual compares two RBAC PolicyRules for semantic equality
func PolicyRulesEqual(rule1, rule2 rbacv1.PolicyRule) bool {
	if SlicesContainSameStrings(rule1.Verbs, rule2.Verbs) &&
		SlicesContainSameStrings(rule1.APIGroups, rule2.APIGroups) &&
		SlicesContainSameStrings(rule1.Resources, rule2.Resources) &&
		SlicesContainSameStrings(rule1.ResourceNames, rule2.ResourceNames) &&
		SlicesContainSameStrings(rule1.NonResourceURLs, rule2.NonResourceURLs) {
		return true
	}
	return false
}

// SliceContainsPolicyRule determines if a given rule is in a slice of rules
func SliceContainsPolicyRule(ruleSlice []rbacv1.PolicyRule, rule rbacv1.PolicyRule) bool {
	for _, r := range ruleSlice {
		if PolicyRulesEqual(rule, r) {
			return true
		}
	}
	return false
}

func ContainerImagePullWait(namespace string, namePrefixes []string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting clientset for kubernetes cluster, error: %v", err))
		return false
	}

	pods, err := ListPodsInCluster(namespace, clientset)
	if err != nil {
		Log(Error, fmt.Sprintf("Error listing pods in cluster for namespace: %s, error: %v", namespace, err))
		return false
	}

	targetPods := []v1.Pod{}
	for _, pod := range pods.Items {
		for _, namePrefix := range namePrefixes {
			if strings.HasPrefix(pod.Name, namePrefix) {
				targetPods = append(targetPods, pod)
			}
		}
	}

	podsWait := make(map[string]bool)
	for _, pod := range targetPods {
		podsWait[pod.Name] = true
	}

	for {
		for _, pod := range pods.Items {
			if _, ok := podsWait[pod.Name]; !ok {
				continue
			}

			containerWaiting := false
			for _, initContainerStatus := range pod.Status.InitContainerStatuses {
				containerWaiting = containerWaiting || (initContainerStatus.State.Waiting != nil)
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				containerWaiting = containerWaiting || (containerStatus.State.Waiting != nil)
			}

			if !containerWaiting {
				delete(podsWait, pod.Name)
			}
		}
		if len(podsWait) == 0 {
			break
		}
	}
	return len(podsWait) == 0
}
