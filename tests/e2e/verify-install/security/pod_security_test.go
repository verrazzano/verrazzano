// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package security

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"regexp"
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
	capDacOverride    = "DAC_OVERRIDE"

	// MySQL ignore pattern
	mysqlPattern = "^mysql-([\\d]+)$"
)

var skipPods = map[string][]string{
	"keycloak": {
		mysqlPattern,
	},
	"verrazzano-install": {
		mysqlPattern,
	},
	"verrazzano-system": {
		"^coherence-operator.*$",
		"^vmi-system-grafana.*$",
		"^weblogic-operator.*$",
	},
	"verrazzano-backup": {
		"^restic.*$",
	},
	"cert-manager": {
		"^external-dns.*$",
	},
}

var skipContainers = []string{"jaeger-collector", "jaeger-query", "jaeger-agent"}
var skipInitContainers = []string{"istio-init", "elasticsearch-init"}

type podExceptions struct {
	allowHostPath    bool
	allowHostNetwork bool
	allowHostPID     bool
	allowHostPort    bool
	containers       map[string]containerException
}

type containerException struct {
	allowedCapabilities map[string]bool
}

var exceptionPods = map[string]podExceptions{
	"node-exporter": {
		allowHostPath:    true,
		allowHostNetwork: true,
		allowHostPID:     true,
		allowHostPort:    true,
	},
	"fluentd": {
		allowHostPath: true,
		containers: map[string]containerException{
			"fluentd": {
				allowedCapabilities: map[string]bool{
					capDacOverride: true,
				},
			},
		},
	},
}

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
			if shouldSkipPod(t.Logs, pod.Name, ns) {
				t.Logs.Debugf("Pod %s/%s on skip list, continuing...", ns, pod.Name)
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
		Entry("Checking pod security in verrazzano-backup", "verrazzano-backup"),
		Entry("Checking pod security in ingress-nginx", "ingress-nginx"),
		Entry("Checking pod security in mysql-operator", "mysql-operator"),
		Entry("Checking pod security in cert-manager", "cert-manager"),
		Entry("Checking pod security in keycloak", "keycloak"),
	)
})

func expectPodSecurityForNamespace(pod corev1.Pod) []error {
	var errors []error

	// get pod exceptions
	isException, exception := isExceptionPod(pod.Name)

	// ensure hostpath is not set unless it is an exception
	if !isException || !exception.allowHostPath {
		for _, vol := range pod.Spec.Volumes {
			if vol.HostPath != nil {
				errors = append(errors, fmt.Errorf("Pod Security not configured for pod %s, HostPath is set, HostPath = %s  Type = %s",
					pod.Name, vol.HostPath.Path, *vol.HostPath.Type))
			}
		}
	}

	// ensure hostnetwork is not set unless it is an exception
	if pod.Spec.HostNetwork && (!isException || !exception.allowHostNetwork) {
		errors = append(errors, fmt.Errorf("Pod Security not configured for pod %s, HostNetwork is set", pod.Name))
	}

	// ensure hostPID is not set unless it is an exception
	if pod.Spec.HostPID && (!isException || !exception.allowHostPID) {
		errors = append(errors, fmt.Errorf("Pod Security not configured for pod %s, HostPID is set", pod.Name))
	}

	// ensure host port is not set unless it is an exception
	if !isException || !exception.allowHostPort {
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if port.HostPort != 0 {
					errors = append(errors, fmt.Errorf("Pod Security not configured for pod %s, HostPort is set, HostPort = %v",
						pod.Name, port.HostPort))
				}
			}
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
	if sc.RunAsNonRoot != nil && !*sc.RunAsNonRoot {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, RunAsNonRoot != true", podName))
	}
	if sc.SeccompProfile == nil {
		errors = append(errors, fmt.Errorf("PodSecurityContext not configured correctly for pod %s, Missing seccompProfile", podName))
	}
	return errors
}

func ensureContainerSecurityContext(sc *corev1.SecurityContext, podName, containerName string) []error {
	exceptionContainer := false
	exceptionPod, exception := isExceptionPod(podName)
	if exceptionPod && exception.containers != nil {
		_, exceptionContainer = exception.containers[containerName]
	}

	if sc == nil {
		return []error{fmt.Errorf("SecurityContext is nil for pod %s, container %s", podName, containerName)}
	}
	var errors []error
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s,  RunAsUser is 0", podName, containerName))
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
	errors = append(errors, checkContainerCapabilities(sc, podName, containerName, exceptionContainer, exception)...)
	return errors
}

func checkContainerCapabilities(sc *corev1.SecurityContext, podName string, containerName string, exceptionContainer bool, exception podExceptions) []error {
	if sc.Capabilities == nil {
		return []error{fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Capabilities is nil", podName, containerName)}
	}
	var errors []error
	dropCapabilityFound := false
	for _, c := range sc.Capabilities.Drop {
		if string(c) == "ALL" {
			dropCapabilityFound = true
		}
	}
	if !dropCapabilityFound {
		errors = append(errors, fmt.Errorf("SecurityContext not configured correctly for pod %s, container %s, Missing `Drop -ALL` capabilities", podName, containerName))
	}
	if len(sc.Capabilities.Add) > 0 && !exceptionContainer {
		if len(sc.Capabilities.Add) > 1 || sc.Capabilities.Add[0] != capNetBindService {
			errors = append(errors, fmt.Errorf("only %s capability allowed, found unexpected capabilities added to container %s in pod %s: %v", capNetBindService, containerName, podName, sc.Capabilities.Add))
		}
	}
	if exceptionContainer && len(sc.Capabilities.Add) > 0 {
		if !capExceptions(exception.containers[containerName].allowedCapabilities, sc.Capabilities.Add) {
			errors = append(errors, fmt.Errorf("%v capabilities are allowed, found unexpected capabilities for pod %s container %s: %v", exception.containers[containerName].allowedCapabilities, podName, containerName, sc.Capabilities.Add))
		}
	}
	return errors
}

func shouldSkipPod(log *zap.SugaredLogger, podName, ns string) bool {
	for _, pattern := range skipPods[ns] {
		podNamePattern := pattern
		match, err := regexp.MatchString(podNamePattern, podName)
		if err != nil {
			log.Errorf("Error parsing regex %s: %s", podNamePattern, err.Error())
		}
		log.Debugf("Matching pod %s against regex %s, result: %v", podName, podNamePattern, match)
		if match {
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

func isExceptionPod(podName string) (bool, podExceptions) {
	for exceptionPod := range exceptionPods {
		if strings.Contains(podName, exceptionPod) {
			return true, exceptionPods[exceptionPod]
		}
	}
	return false, podExceptions{}
}

func capExceptions(allowedCaps map[string]bool, givenCaps []corev1.Capability) bool {
	exceptionsOk := true
	for _, givenCap := range givenCaps {
		_, ok := allowedCaps[string(givenCap)]
		exceptionsOk = exceptionsOk && ok
	}
	return exceptionsOk
}
