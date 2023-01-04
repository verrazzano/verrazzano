// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var t = framework.NewTestFramework("security")

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second

	// Only allowed capability in restricted mode
	capNetBindService = "NET_BIND_SERVICE"
)

var skipPods = map[string][]string{
	"verrazzano-install": {},
	"verrazzano-system": {
		"coherence-operator",
		"fluentd",
		"oam-kubernetes-runtime",
		"vmi-system",
		"weblogic-operator",
	},
	"verrazzano-monitoring": {
		"node-exporter",
		"alertmanager",
		"pushgateway",
		"kube-state-metrics",
		"prometheus-adapter",
		"jaeger-operator",
	},
}

var skipContainers = []string{}
var skipInitContainers = []string{"istio-init"}

var (
	clientset *kubernetes.Clientset
)

var isMinVersion150 bool

var beforeSuite = t.BeforeSuiteFunc(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	isMinVersion150, err = pkg.IsVerrazzanoMinVersion("1.5.0", kubeconfigPath)
	if err != nil {
		Fail(fmt.Sprintf("Error checking Verrazzano version: %s", err.Error()))
	}
	Eventually(func() (*kubernetes.Clientset, error) {
		clientset, err = k8sutil.GetKubernetesClientset()
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = BeforeSuite(beforeSuite)

var _ = t.AfterEach(func() {})

var _ = t.Describe("Ensure pod security", Label("f:security.podsecurity"), func() {
	testFunc := func(ns string) {
		if !isMinVersion150 {
			t.Logs.Infof("Skipping test, minimum version requirement of v1.5.0 not met")
			return
		}
		var podList *corev1.PodList
		Eventually(func() (*corev1.PodList, error) {
			var err error
			podList, err = clientset.CoreV1().Pods(ns).List(context.TODO(), v1.ListOptions{})
			return podList, err
		}, waitTimeout, pollingInterval).ShouldNot(BeNil())

		t.Logs.Debugf("Podlist length: %v", len(podList.Items))

		var errors []error
		pods := podList.Items
		for _, pod := range pods {
			t.Logs.Debugf("Checking pod %s/%s", ns, pod.Name)
			if shouldSkipPod(pod.Name, ns) {
				continue
			}
			errors = append(errors, expectPodSecurityForNamespace(pod)...)
		}
		Expect(errors).To(BeEmpty())
	}
	t.DescribeTable("Check pod security in system namespaces", testFunc,
		Entry("Checking pod security in verrazzano-install", "verrazzano-install"),
		Entry("Checking pod security in verrazzano-system", "verrazzano-system"),
		Entry("Checking pod security in verrazzano-monitoring", "verrazzano-monitoring"),
	)
})

func expectPodSecurityForNamespace(pod corev1.Pod) []error {
	var errors []error
	// ensure hostpath is not set
	for _, vol := range pod.Spec.Volumes {
		if vol.HostPath != nil {
			errors = append(errors, fmt.Errorf("Pod Security not configured for pod %s, HostPath is set, HostPath = %s  Type = %s",
				pod.Name, vol.HostPath.Path, *vol.HostPath.Type))
		}
	}

	// ensure pod SecurityContext set correctly
	if errs := ensurePodSecurityContext(pod.Spec.SecurityContext, pod.Name); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// ensure container SecurityContext set correctly
	for _, container := range pod.Spec.Containers {
		if shouldSkipContainer(container.Name, skipContainers) {
			continue
		}
		if errs := ensureContainerSecurityContext(container.SecurityContext, pod.Name, container.Name); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// ensure init container SecurityContext set correctly
	for _, initContainer := range pod.Spec.InitContainers {
		if shouldSkipContainer(initContainer.Name, skipInitContainers) {
			continue
		}
		if errs := ensureContainerSecurityContext(initContainer.SecurityContext, pod.Name, initContainer.Name); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}
	return errors
}

func ensurePodSecurityContext(sc *corev1.PodSecurityContext, podName string) []error {
	if sc == nil {
		return []error{fmt.Errorf("PodSecurityContext is nil for pod %s", podName)}
	}
	var errors []error
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsUser is 0", podName))
	}
	if sc.RunAsGroup != nil && *sc.RunAsGroup == 0 {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsGroup is 0", podName))
	}
	if sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsNonRoot != true", podName))
	}
	if sc.SeccompProfile == nil {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, Missing seccompProfile", podName))
	}
	return errors
}

func ensureContainerSecurityContext(sc *corev1.SecurityContext, podName, containerName string) []error {
	if sc == nil {
		return []error{fmt.Errorf("SecurityContext is nil for pod %s, container %s", podName, containerName)}
	}
	var errors []error
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s,  RunAsUser is 0", podName, containerName))
	}
	if sc.RunAsGroup != nil && *sc.RunAsGroup == 0 {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsGroup is 0", podName, containerName))
	}
	if sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsNonRoot != true", podName, containerName))
	}
	if sc.Privileged == nil || *sc.Privileged {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Privileged != false", podName, containerName))
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, AllowPrivilegeEscalation != false", podName, containerName))
	}
	if sc.Capabilities == nil {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Capabilities is nil", podName, containerName))
	}
	dropCapabilityFound := false
	for _, c := range sc.Capabilities.Drop {
		if string(c) == "ALL" {
			dropCapabilityFound = true
		}
	}
	if !dropCapabilityFound {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Missing `Drop -ALL` capabilities", podName, containerName))
	}
	if len(sc.Capabilities.Add) > 0 {
		if len(sc.Capabilities.Add) > 1 || sc.Capabilities.Add[0] != capNetBindService {
			errors = append(errors, fmt.Errorf("only %s capability allowed, found unexpected capabilities added to container %s in pod %s: %v", capNetBindService, containerName, podName, sc.Capabilities.Add))
		}
	}
	return errors
}

func shouldSkipPod(podName, ns string) bool {
	for _, pod := range skipPods[ns] {
		if strings.Contains(podName, pod) {
			return true
		}
	}
	return false
}

func shouldSkipContainer(containerName string, skip []string) bool {
	for _, c := range skip {
		if strings.Contains(containerName, c) {
			return true
		}
	}
	return false
}
