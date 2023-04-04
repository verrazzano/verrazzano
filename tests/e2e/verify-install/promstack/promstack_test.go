// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package promstack

import (
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout                     = 3 * time.Minute
	pollingInterval                 = 10 * time.Second
	prometheusTLSSecret             = "prometheus-operator-kube-p-admission"
	prometheusOperatorDeployment    = "prometheus-operator-kube-p-operator"
	prometheusOperatorContainerName = "kube-prometheus-stack"
	thanosSidecarContainerName      = "thanos-sidecar"
	overrideConfigMapSecretName     = "test-overrides"
	overrideKey                     = "override"
	overrideValue                   = "true"
)

type enabledFunc func(string) bool

type enabledComponents struct {
	podName     string
	enabledFunc enabledFunc
	target      string
}

var (
	promStackEnabledComponents = []enabledComponents{
		{podName: "prometheus-adapter", enabledFunc: pkg.IsPrometheusAdapterEnabled},
		{podName: "prometheus-operator-kube-p-operator", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{podName: "kube-state-metrics", enabledFunc: pkg.IsKubeStateMetricsEnabled},
		{podName: "prometheus-pushgateway", enabledFunc: pkg.IsPrometheusPushgatewayEnabled},
		{podName: "prometheus-node-exporter", enabledFunc: pkg.IsPrometheusNodeExporterEnabled},
		{podName: "prometheus-prometheus-operator-kube-p-prometheus", enabledFunc: pkg.IsPrometheusEnabled},
	}
	promOperatorCrds = []string{
		"alertmanagerconfigs.monitoring.coreos.com",
		"alertmanagers.monitoring.coreos.com",
		"podmonitors.monitoring.coreos.com",
		"probes.monitoring.coreos.com",
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"servicemonitors.monitoring.coreos.com",
		"thanosrulers.monitoring.coreos.com",
	}
	imagePrefix              = pkg.GetImagePrefix()
	expectedPromOperatorArgs = []string{
		"--prometheus-default-base-image=" + imagePrefix + "/verrazzano/prometheus",
		"--alertmanager-default-base-image=" + imagePrefix + "/verrazzano/alertmanager",
	}
	labelMatch      = map[string]string{overrideKey: overrideValue}
	isMinVersion140 bool

	//scrapeTargetsMap = map[string]func(cr runtime.Object) bool{
	//	"serviceMonitor/verrazzano-monitoring/authproxy":                             vzcr.IsAuthProxyEnabled,
	//	"serviceMonitor/verrazzano-monitoring/fluentd":                               vzcr.IsFluentdEnabled,
	//	"serviceMonitor/verrazzano-monitoring/kube-state-metrics":                    vzcr.IsKubeStateMetricsEnabled,
	//	"serviceMonitor/verrazzano-monitoring/opensearch":                            vzcr.IsOpenSearchEnabled,
	//	"serviceMonitor/verrazzano-monitoring/pilot":                                 vzcr.IsIstioEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-node-exporter":              vzcr.IsNodeExporterEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-apiserver":  vzcr.IsPrometheusOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-coredns":    vzcr.IsPrometheusOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-kube-proxy": vzcr.IsPrometheusOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-kubelet":    vzcr.IsPrometheusOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-operator":   vzcr.IsPrometheusOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-prometheus": vzcr.IsPrometheusEnabled,
	//	"serviceMonitor/verrazzano-monitoring/verrazzano-application-operator":       vzcr.IsApplicationOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/verrazzano-cluster-operator":           vzcr.IsClusterOperatorEnabled,
	//	"serviceMonitor/verrazzano-monitoring/verrazzano-monitoring-operator":        vzcr.IsVMOEnabled,
	//	"serviceMonitor/verrazzano-monitoring/verrazzano-platform-operator":          func(_ runtime.Object) bool { return true },
	//}

	scrapeTargets = []enabledComponents{
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-node-exporter", enabledFunc: pkg.IsPrometheusNodeExporterEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-apiserver", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-coredns", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-kubelet", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-operator", enabledFunc: pkg.IsPrometheusOperatorEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/prometheus-operator-kube-p-prometheus", enabledFunc: pkg.IsPrometheusEnabled},
		{target: "serviceMonitor/verrazzano-monitoring/verrazzano-platform-operator", enabledFunc: func(_ string) bool { return true }},
	}
)

var t = framework.NewTestFramework("promstack")

func listEnabledComponents() []string {
	kubeconfigPath := getKubeConfigOrAbort()
	var enabledPods []string
	for _, component := range promStackEnabledComponents {
		if component.enabledFunc(kubeconfigPath) {
			enabledPods = append(enabledPods, component.podName)
		}
	}
	return enabledPods
}

func listDisabledComponents() []string {
	kubeconfigPath := getKubeConfigOrAbort()
	var disabledPods []string
	for _, component := range promStackEnabledComponents {
		if !component.enabledFunc(kubeconfigPath) {
			disabledPods = append(disabledPods, component.podName)
		}
	}
	return disabledPods
}

func isPrometheusOperatorEnabled() bool {
	return pkg.IsPrometheusOperatorEnabled(getKubeConfigOrAbort())
}

// areOverridesEnabled - return true if the override value prometheusOperator.podAnnotations.override
// is present and set to "true"
func areOverridesEnabled() bool {
	kubeconfigPath := getKubeConfigOrAbort()
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get vz resource in cluster: %s", err.Error()))
		return false
	}

	promOper := vz.Spec.Components.PrometheusOperator
	if promOper == nil || len(promOper.ValueOverrides) == 0 {
		return false
	}

	// The overrides are enabled if the override value prometheusOperator.podAnnotations.override = "true"
	for _, override := range promOper.ValueOverrides {
		if override.Values != nil {
			jsonString, err := gabs.ParseJSON(override.Values.Raw)
			if err != nil {
				return false
			}
			if container := jsonString.Path("prometheusOperator.podAnnotations.override"); container != nil {
				if val, ok := container.Data().(string); ok {
					return "true" == string(val)
				}
			}
		}
	}

	return false
}

// isThanosSidecarEnabled returns true if the Helm override for enabling the Thanos sidecar is found
func isThanosSidecarEnabled() (bool, error) {
	kubeconfigPath := getKubeConfigOrAbort()
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Failed to get installed Verrazzano resource in the cluster: %v", err)
		return false, err
	}

	if vz.Spec.Components.PrometheusOperator == nil {
		return false, nil
	}
	overrides := vz.Spec.Components.PrometheusOperator.ValueOverrides

	for _, override := range overrides {
		if override.Values == nil {
			continue
		}
		vals, err := gabs.ParseJSON(override.Values.Raw)
		if err != nil {
			t.Logs.Errorf("Failed to parse the Values Override JSON: %v", err)
			return false, err
		}

		integration, ok := vals.Path("prometheus.thanos.integration").Data().(string)
		t.Logs.Debugf("Integration Override: %s", integration)
		if ok && integration == "sidecar" {
			return true, nil
		}
	}
	t.Logs.Infof("Thanos Sidecar override not found in the Prometheus Operator overrides.")
	return false, nil
}

// isThanosSidecarInstalledIfEnabled returns true if Thanos is disabled, or if Thanos is enabled and the sidecar container is found
func isThanosSidecarInstalledIfEnabled() (bool, error) {
	enabled, err := isThanosSidecarEnabled()
	if err != nil {
		return false, err
	}
	if !enabled {
		return true, nil
	}

	promPod, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app.kubernetes.io/name": "prometheus"},
	}, constants.VerrazzanoMonitoringNamespace)
	if err != nil {
		t.Logs.Errorf("Failed to get the Prometheus pod from the cluster: %v", err)
		return false, err
	}

	for _, pod := range promPod {
		for _, container := range pod.Spec.Containers {
			if container.Name == thanosSidecarContainerName {
				return true, nil
			}
		}
	}
	return false, nil
}

// 'It' Wrapper to only run spec if the Prometheus Stack is supported on the current Verrazzano version
func WhenPromStackInstalledIt(description string, f func()) {
	kubeconfigPath := getKubeConfigOrAbort()
	supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.3.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Prometheus stack is not supported", description)
	}
}

func getKubeConfigOrAbort() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeconfigPath
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	kubeconfigPath := getKubeConfigOrAbort()
	isMinVersion140, err = pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Prometheus Stack", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenPromStackInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				if len(listEnabledComponents()) == 0 {
					return true, nil
				}
				return pkg.DoesNamespaceExist(constants.VerrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the pods are running
		// THEN we successfully find the running pods
		WhenPromStackInstalledIt("should have running pods", func() {
			promStackPodsRunning := func() (bool, error) {
				enabledPods := listEnabledComponents()
				result, err := pkg.PodsRunning(constants.VerrazzanoMonitoringNamespace, enabledPods)
				if err != nil {
					t.Logs.Errorf("Pods %v is not running in the namespace: %v, error: %v", enabledPods, constants.VerrazzanoMonitoringNamespace, err)
				}
				return result, err
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure any disabled pods are NOT running
		// THEN we successfully find the running pods
		WhenPromStackInstalledIt("Disabled pods not running", func() {
			promStackPodsRunning := func() bool {
				disabledPods := listDisabledComponents()
				t.Logs.Debugf("Checking disabled component pods %v", disabledPods)
				result, err := pkg.PodsNotRunning(constants.VerrazzanoMonitoringNamespace, disabledPods)
				if err != nil {
					AbortSuite(fmt.Sprintf("Unexpected error occurred checking for pods %v in namespace: %v, error: %v", disabledPods, constants.VerrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the operator overrides get applied
		// THEN we see that the correct pod labels and annotations exist
		WhenPromStackInstalledIt("should have Prometheus Operator pod labeled and annotated", func() {
			promStackPodsRunning := func() bool {
				if isPrometheusOperatorEnabled() && areOverridesEnabled() {
					_, err := pkg.GetConfigMap(overrideConfigMapSecretName, constants.DefaultNamespace)
					if err == nil {
						pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{
							MatchLabels: labelMatch,
						}, constants.VerrazzanoMonitoringNamespace)
						if err != nil {
							AbortSuite(fmt.Sprintf("Label override not found for the Prometheus Operator pod in namespace %s: %v", constants.VerrazzanoMonitoringNamespace, err))
						}
						foundAnnotation := false
						for _, pod := range pods {
							if val, ok := pod.Annotations[overrideKey]; ok && val == overrideValue {
								foundAnnotation = true
							}
						}
						return len(pods) == 1 && foundAnnotation
					} else if !k8serrors.IsNotFound(err) {
						AbortSuite(fmt.Sprintf("Error retrieving the override ConfigMap: %v", err))
					}
				}
				return true
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN we check to make sure the default images are from Verrazzano
		// THEN we see that the arguments are correctly populated
		WhenPromStackInstalledIt(fmt.Sprintf("should have the correct default image arguments: %s, %s", expectedPromOperatorArgs[0], expectedPromOperatorArgs[1]), func() {
			promStackPodsRunning := func() (bool, error) {
				if isPrometheusOperatorEnabled() {
					return pkg.ContainerHasExpectedArgs(constants.VerrazzanoMonitoringNamespace, prometheusOperatorDeployment, prometheusOperatorContainerName, expectedPromOperatorArgs)
				}
				return true, nil
			}
			Eventually(promStackPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPromStackInstalledIt("should have the correct Prometheus Operator CRDs", func() {
			verifyCRDList := func() (bool, error) {
				if isPrometheusOperatorEnabled() {
					for _, crd := range promOperatorCrds {
						exists, err := pkg.DoesCRDExist(crd)
						if err != nil || !exists {
							return exists, err
						}
					}
					return true, nil
				}
				return true, nil
			}
			Eventually(verifyCRDList, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Prometheus stack is installed
		// WHEN the Prometheus Instance has Thanos enabled
		// THEN we see that the Thanos sidecar exists
		WhenPromStackInstalledIt("should have the Thanos sidecar if enabled", func() {
			Eventually(func() (bool, error) {
				return isThanosSidecarInstalledIfEnabled()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPromStackInstalledIt("should have the TLS secret", func() {
			Eventually(func() bool {
				if isPrometheusOperatorEnabled() {
					return pkg.SecretsCreated(constants.VerrazzanoMonitoringNamespace, prometheusTLSSecret)
				}
				return true
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenPromStackInstalledIt("has affinity configured on prometheus pods", func() {
			if isMinVersion140 {
				var pods []corev1.Pod
				Eventually(func() error {
					var err error
					selector := map[string]string{
						"prometheus":             "prometheus-operator-kube-p-prometheus",
						"app.kubernetes.io/name": "prometheus",
					}
					pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: selector}, constants.VerrazzanoMonitoringNamespace)
					return err
				}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

				// Check the affinity configuration. Verify only a pod anti-affinity definition exists.
				for _, pod := range pods {
					affinity := pod.Spec.Affinity
					Expect(affinity).ToNot(BeNil())
					Expect(affinity.PodAffinity).To(BeNil())
					Expect(affinity.NodeAffinity).To(BeNil())
					Expect(affinity.PodAntiAffinity).ToNot(BeNil())
					Expect(len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
				}
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not v1.4.0")
			}
		})

		WhenPromStackInstalledIt("should have scrape targets healthy for installed components", func() {
			if isMinVersion140 && !pkg.IsManagedClusterProfile() {
				verifyScrapeTargets := func() (bool, error) {
					var targets []string
					for _, scrapeTarget := range scrapeTargets {
						if scrapeTarget.enabledFunc(getKubeConfigOrAbort()) {
							targets = append(targets, scrapeTarget.target)
						}
					}
					if pkg.IsPrometheusEnabled(getKubeConfigOrAbort()) {
						if !pkg.IsIngressEnabled(getKubeConfigOrAbort()) {
							return pkg.ScrapeTargetsHealthyFromExec(targets)
						}
						return pkg.ScrapeTargetsHealthy(targets)
					}
					return true, nil
				}
				Eventually(verifyScrapeTargets, waitTimeout, pollingInterval).Should(BeTrue())
			} else {
				t.Logs.Info("Skipping check, Verrazzano minimum version is not v1.4.0 or is managed cluster")
			}
		})
	})
})
