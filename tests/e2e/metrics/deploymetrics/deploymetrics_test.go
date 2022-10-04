// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deploymetrics

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deploymetricsCompYaml   = "testdata/deploymetrics/deploymetrics-comp.yaml"
	deploymetricsCompName   = "deploymetrics-deployment"
	deploymetricsAppYaml    = "testdata/deploymetrics/deploymetrics-app.yaml"
	deploymetricsAppName    = "deploymetrics-appconf"
	skipVerifications       = "Skip Verifications"
	errGenerateSvcMonJobFmt = "Failed to generate the Service Monitor job name: %v"
)

var (
	expectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
	generatedNamespace           = pkg.GenerateNamespace("deploymetrics")

	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	longWaitTimeout          = 15 * time.Minute
	longPollingInterval      = 30 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	clusterNameLabel string

	t = framework.NewTestFramework("deploymetrics")
)

var clusterDump = pkg.NewClusterDumpWrapper(generatedNamespace)
var kubeconfig string
var _ = clusterDump.BeforeSuite(func() {
	if !skipDeploy {
		deployMetricsApplication()
	}
	var err error

	kubeconfig, err = getKubeConfigPath()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get the kubeconfig: %v", err))
	}
	clusterNameLabel, err = pkg.GetClusterNameMetricLabel(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		Fail(err.Error())
	}

	// Create a service in VZ versions 1.4.0 and greater, so that a servicemonitor will be generated
	// by Verrazzano for Prometheus Operator
	isVzMinVer14, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfig)
	if isVzMinVer14 {
		Eventually(func() error {
			promJobName, err := getPromJobName()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf(errGenerateSvcMonJobFmt, err))
			}
			serviceName := promJobName
			err = createService(serviceName)
			// if this is running post upgrade, we may have already run this test pre-upgrade and
			// created the service. It is not an error if the service already exists.
			if err != nil && !errors.IsAlreadyExists(err) {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to create the Service for the Service Monitor: %v", err))
				return err
			}
			return nil
		}, waitTimeout, pollingInterval).Should(BeNil(), "Expected to be able to create the metrics service")
	}
})
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	if !skipUndeploy {
		undeployMetricsApplication()
	}
})

func deployMetricsApplication() {
	t.Logs.Info("Deploy DeployMetrics Application")

	t.Logs.Info("Create namespace")
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(deploymetricsCompYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(deploymetricsAppYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create DeployMetrics application resource")

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsDeploymetricsApp)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())

	t.Logs.Info("Verify deploymetrics-workload pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsDeploymetricsApp)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, waitTimeout, pollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployMetricsApplication() {
	t.Logs.Info("Undeploy DeployMetrics Application")

	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(deploymetricsCompYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(deploymetricsAppYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pods to terminate")
	Eventually(func() bool {
		podsNotRunning, _ := pkg.PodsNotRunning(namespace, expectedPodsDeploymetricsApp)
		return podsNotRunning
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	isVzMinVer14, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfig)
	if isVzMinVer14 {
		Eventually(func() bool {
			serviceName, err := getPromJobName()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf(errGenerateSvcMonJobFmt, err))
				return false
			}
			_, err = pkg.GetServiceMonitor(namespace, serviceName)
			return err != nil
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Service Monitor to not exist")
	}
	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, longWaitTimeout, longPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Waiting for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("DeployMetrics Application test", Label("f:app-lcm.oam"), func() {

	t.Context("for Prometheus Config.", Label("f:observability.monitoring.prom"), func() {
		t.It("Verify that Prometheus Service Monitor exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			isVzMinVer14, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfig)
			if isVzMinVer14 {
				serviceName, err := getPromJobName()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf(errGenerateSvcMonJobFmt, err))
				}
				Eventually(func() (*promoperapi.ServiceMonitor, error) {
					monitor, err := pkg.GetServiceMonitor(namespace, serviceName)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Failed to get the Service Monitor from the cluster: %v", err))
					}
					return monitor, err
				}, waitTimeout, pollingInterval).Should(Not(BeNil()), "Expected to find Service Monitor")
			}
		})
	})

	t.Context("Retrieve Prometheus scraped metrics for", Label("f:observability.monitoring.prom"), func() {
		t.It("App Component", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterName, err := getClusterNameForPromQuery()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed getting the cluster name: %v", err))
			}
			metricLabels := map[string]string{
				"app_oam_dev_name": "deploymetrics-appconf",
				clusterNameLabel:   clusterName,
			}
			Eventually(func() bool {
				return pkg.MetricsExistInCluster("http_server_requests_seconds_count", metricLabels, kubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		t.It("App Config", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterName, err := getClusterNameForPromQuery()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed getting the cluster name: %v", err))
			}
			metricLabels := map[string]string{
				"app_oam_dev_component": "deploymetrics-deployment",
				clusterNameLabel:        clusterName,
			}
			Eventually(func() bool {
				return pkg.MetricsExistInCluster("tomcat_sessions_created_sessions_total", metricLabels, kubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})

})

// Return the cluster name used for the Prometheus query
func getClusterNameForPromQuery() (string, error) {
	if pkg.IsManagedClusterProfile() {
		var clusterName = os.Getenv("CLUSTER_NAME")
		pkg.Log(pkg.Info, "This is a managed cluster, returning cluster name - "+clusterName)
		return clusterName, nil
	}
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return kubeConfigPath, err
	}
	isMinVersion110, err := pkg.IsVerrazzanoMinVersion("1.1.0", kubeConfigPath)
	if err != nil {
		return "", err
	}
	if isMinVersion110 {
		return "local", nil
	}
	return "", nil
}

func getDefaultKubeConfigPath() (string, error) {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return kubeConfigPath, err
	}
	pkg.Log(pkg.Info, "Default kube config path - "+kubeConfigPath)
	return kubeConfigPath, nil
}

func getKubeConfigPath() (string, error) {
	adminKubeConfig, present := os.LookupEnv("ADMIN_KUBECONFIG")
	if pkg.IsManagedClusterProfile() {
		if !present {
			return adminKubeConfig, fmt.Errorf("Environment variable ADMIN_KUBECONFIG is required to run the test")
		}
	} else {
		return getDefaultKubeConfigPath()
	}
	return adminKubeConfig, nil
}

func getPromJobName() (string, error) {
	kubeconfig, err := getKubeConfigPath()
	if err != nil {
		return kubeconfig, err
	}
	usesServiceMonitor, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfig)
	if err != nil {
		return kubeconfig, err
	}
	if usesServiceMonitor {
		return pkg.GetAppServiceMonitorName(namespace, deploymetricsAppName, "deploymetrics-deployment"), nil
	}
	// For VZ versions prior to 1.4.0, the job name in prometheus scrape config was of the old format
	// <app_name>_default_<app_namespace>_<app_component_name>
	return fmt.Sprintf("%s_default_%s_%s", deploymetricsAppName, generatedNamespace, deploymetricsCompName), nil
}

// create Service creates a service for metrics collection
func createService(name string) error {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	cli, err := k8sclient.New(config, k8sclient.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "deploymetrics",
			},
			Ports: []v1.ServicePort{
				{
					Name:     "metrics",
					Port:     8080,
					Protocol: "TCP",
				},
			},
		},
	}
	return cli.Create(context.TODO(), &svc)
}
