package security

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

var t = framework.NewTestFramework("security")

const (
	waitTimeout     = 1 * time.Minute
	pollingInterval = 2 * time.Second
)

var skipPods = map[string][]string{
	"verrazzano-install": {},
	"verrazzano-system": {
		"coherence-operator",
		"fluentd",
		"oam-kubernetes-runtime",
		"verrazzano-authproxy",
		"vmi-system",
		"weblogic-operator",
	},
}

var skipContainers = []string{"istio-proxy"}
var skipInitContainers = []string{"istio-init"}

var (
	kubeconfigPath string
	clientset      *kubernetes.Clientset
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	Eventually(func() (*kubernetes.Clientset, error) {
		clientset, err = k8sutil.GetKubernetesClientset()
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = BeforeSuite(beforeSuite)

var _ = t.AfterEach(func() {})

var _ = t.Describe("Ensure pod security", Label("f:security.podsecurity"), func() {
	for ns := range skipPods {
		t.ItMinimumVersion(fmt.Sprintf("for pods in namespace %s", ns), "1.5.0", kubeconfigPath, func() {
			var podList *corev1.PodList

			Eventually(func() (*corev1.PodList, error) {
				podList, err := clientset.CoreV1().Pods(ns).List(context.TODO(), v1.ListOptions{})
				return podList, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())

			pods := podList.Items
			for _, pod := range pods {
				if shouldSkipPod(pod.Name, ns) {
					continue
				}
				Expect(expectPodSecurityForNamespace(&pod)).To(BeTrue())
			}
		})
	}
})

func expectPodSecurityForNamespace(pod *corev1.Pod) bool {
	// ensure hostpath is not set
	for _, vol := range pod.Spec.Volumes {
		if vol.HostPath != nil {
			t.Logs.Errorf("Pod Security not configured for pod %, HostPath is set, HostPath = %s  Type = %s",
				pod.Name, vol.HostPath.Path, *vol.HostPath.Type)
			return false
		}
	}
	// ensure pod SecurityContext set correctly
	if err := ensurePodSecurityContext(pod.Spec.SecurityContext, pod.Name); err != nil {
		t.Logs.Error(err)
		return false
	}
	for _, container := range pod.Spec.Containers {
		if err := ensureContainerSecurityContext(container.SecurityContext, pod.Name, container.Name); err != nil {
			t.Logs.Error(err)
			return false
		}
	}
	for _, initContainer := range pod.Spec.InitContainers {
		if err := ensureContainerSecurityContext(initContainer.SecurityContext, pod.Name, initContainer.Name); err != nil {
			t.Logs.Error(err)
			return false
		}
	}

	return true
}

func ensurePodSecurityContext(sc *corev1.PodSecurityContext, podName string) error {
	if sc == nil {
		return fmt.Errorf("PodSecurityContext is nil for pod %s", podName)
	}
	if *sc.RunAsUser == 0 {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsUser is 0", podName)
	}
	if *sc.RunAsGroup == 0 {
		return fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsGroup is 0", podName)
	}
	if !*sc.RunAsNonRoot {
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
	if *sc.RunAsUser == 0 {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s,  RunAsUser is 0", podName, containerName)
	}
	if *sc.RunAsGroup == 0 {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsGroup is 0", podName, containerName)
	}
	if !*sc.RunAsNonRoot {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, RunAsNonRoot != true", podName, containerName)
	}
	if *sc.Privileged {
		return fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Privileged != false", podName, containerName)
	}
	if *sc.AllowPrivilegeEscalation {
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
