// Copyright (c) 2022, Oracle and/or its affiliates.
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
	overrideConfigMapSecretName     = "test-overrides"
	overrideKey                     = "override"
	overrideValue                   = "true"
)

type enabledFunc func(string) bool

type enabledComponents struct {
	podName     string
	enabledFunc enabledFunc
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
)

var t = framework.NewTestFramework("promstack")

func listEnabledComponents() []string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	var enabledPods []string
	for _, component := range promStackEnabledComponents {
		if component.enabledFunc(kubeconfigPath) {
			enabledPods = append(enabledPods, component.podName)
		}
	}
	return enabledPods
}

func isPrometheusOperatorEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return pkg.IsPrometheusOperatorEnabled(kubeconfigPath)
}

// areOverridesEnabled - return true if the override value prometheusOperator.podAnnotations.override
// is present and set to "true"
func areOverridesEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
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

// 'It' Wrapper to only run spec if the Prometheus Stack is supported on the current Verrazzano version
func WhenPromStackInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
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

var _ = t.BeforeSuite(func() {
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	isMinVersion140, err = pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
})

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
			promStackPodsRunning := func() bool {
				enabledPods := listEnabledComponents()
				result, err := pkg.PodsRunning(constants.VerrazzanoMonitoringNamespace, enabledPods)
				if err != nil {
					AbortSuite(fmt.Sprintf("Pods %v is not running in the namespace: %v, error: %v", enabledPods, constants.VerrazzanoMonitoringNamespace, err))
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
	})
})
