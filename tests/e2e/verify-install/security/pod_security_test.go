// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package security

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var t = framework.NewTestFramework("security")

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
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
}

var skipContainers = []string{"istio-proxy"}
var skipInitContainers = []string{"istio-init"}

var (
	clientset *kubernetes.Clientset
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	Eventually(func() (*kubernetes.Clientset, error) {
		clientset, err := k8sutil.GetKubernetesClientset()
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = BeforeSuite(beforeSuite)

var _ = t.AfterEach(func() {})

var _ = t.Describe("Ensure pod security", Label("f:security.podsecurity"), func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	for ns := range skipPods {
		t.ItMinimumVersion(fmt.Sprintf("for pods in namespace %s", ns), "1.5.0", kubeconfigPath, func() {
			var podList *corev1.PodList
			var err error
			Eventually(func() (*corev1.PodList, error) {
				podList, err = clientset.CoreV1().Pods(ns).List(context.TODO(), v1.ListOptions{})
				return podList, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			pods := podList.Items
			for _, pod := range pods {
				if shouldSkipPod(pod.Name, ns) {
					continue
				}
				Expect(expectPodSecurityForNamespace(pod)).To(Not(HaveOccurred()))
			}
		})
		t.Logs.Infof("Pod security verified for namespace %s", ns)
	}
})

func expectPodSecurityForNamespace(pod corev1.Pod) error {
	// ensure hostpath is not set
	for _, vol := range pod.Spec.Volumes {
		if vol.HostPath != nil {
			return fmt.Errorf("Pod Security not configured for pod %s, HostPath is set, HostPath = %s  Type = %s",
				pod.Name, vol.HostPath.Path, *vol.HostPath.Type)
		}
	}

	// ensure pod SecurityContext set correctly
	if err := ensurePodSecurityContext(pod.Spec.SecurityContext, pod.Name); err != nil {
		return err
	}

	// ensure container SecurityContext set correctly
	for _, container := range pod.Spec.Containers {
		if shouldSkipContainer(container.Name, skipContainers) {
			continue
		}
		if err := ensureContainerSecurityContext(container.SecurityContext, pod.Name, container.Name); err != nil {
			return err
		}
	}

	// ensure init container SecurityContext set correctly
	for _, initContainer := range pod.Spec.InitContainers {
		if shouldSkipContainer(initContainer.Name, skipInitContainers) {
			continue
		}
		if err := ensureContainerSecurityContext(initContainer.SecurityContext, pod.Name, initContainer.Name); err != nil {
			return err
		}
	}

	return nil
}

func ensurePodSecurityContext(sc *corev1.PodSecurityContext, podName string) error {
	if sc == nil {
		return fmt.Errorf("PodSecurityContext is nil for pod %s", podName)
	}
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsUser is 0", podName)
	}
	if sc.RunAsGroup != nil && *sc.RunAsGroup == 0 {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsGroup is 0", podName)
	}
	if sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsNonRoot != true", podName)
	}
	if sc.SeccompProfile == nil {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, Missing seccompProfile", podName)
	}
	return nil
}

func ensureContainerSecurityContext(sc *corev1.SecurityContext, podName, containerName string) error {
	if sc == nil {
		return fmt.Errorf("SecurityContext is nil for pod %s, container %s", podName, containerName)
	}
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s,  RunAsUser is 0", podName, containerName)
	}
	if sc.RunAsGroup != nil && *sc.RunAsGroup == 0 {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsGroup is 0", podName, containerName)
	}
	if sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsNonRoot != true", podName, containerName)
	}
	if sc.Privileged == nil || *sc.Privileged {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Privileged != false", podName, containerName)
	}
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, AllowPrivilegeEscalation != false", podName, containerName)
	}
	if sc.Capabilities == nil {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Capabilities is nil", podName, containerName)
	}
	dropCapabilityFound := false
	for _, c := range sc.Capabilities.Drop {
		if string(c) == "ALL" {
			dropCapabilityFound = true
		}
	}
	if !dropCapabilityFound {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Missing `Drop -ALL` capabilities", podName, containerName)
	}

	return nil
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
