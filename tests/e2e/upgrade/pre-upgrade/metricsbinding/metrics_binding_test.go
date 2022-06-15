// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	// Define the test namespaces
	deploymentNamespace  = "hello-helidon-deployment"
	podNamespace         = "hello-helidon-pod"
	replicasetNamespace  = "hello-helidon-replicaset"
	statefulsetNamespace = "hello-helidon-statefulset"

	// Define the test deployment name prefix
	namePrefix = "hello-helidon-"

	// Define the test yaml file locations
	deploymentYaml  = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-deployment.yaml"
	podYaml         = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-pod.yaml"
	replicasetYaml  = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-replicaset.yaml"
	statefulsetYaml = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-statefulset.yaml"

	// Define the custom templates
	legacyVMITemplate    = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/legacy-vmi-metrics-template.yaml"
	externalPromTemplate = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/external-prometheus-metrics-template.yaml"

	// Define the simulated external Prometheus ConfigMap
	configMapYaml = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/external-prometheus-config.yaml"
)

var (
	t           = framework.NewTestFramework("deploymentworkload")
	clusterDump = pkg.NewClusterDumpWrapper()
)

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.AfterEach(func() {})

// 'It' Wrapper to only run spec if the Prometheus Stack is supported on the current Verrazzano version
func WhenMetricsBindingInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoBelowVersion("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version less than 1.4.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Metrics Binding is not supported", description)
	}
}

var _ = t.Describe("Verify", Label("f:app-lcm.poko"), func() {

	// GIVEN the cluster version
	// WHEN the Prometheus metrics in the app namespace are scraped
	// THEN the Helidon application metrics should exist using the default metrics template for deployments
	t.Context("Deploy and verify the test applications", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		WhenMetricsBindingInstalledIt("Apply the Deployment and external Prometheus Metrics Template", func() {
			DeployConfigMap(deploymentNamespace, configMapYaml, *t)
			DeployTemplate(deploymentNamespace, externalPromTemplate, *t)
			DeployApplication(deploymentNamespace, deploymentYaml, namePrefix, "enabled", *t)
		})
		WhenMetricsBindingInstalledIt("Apply the Pod and legacy VMI Metrics Template", func() {
			DeployTemplate(podNamespace, legacyVMITemplate, *t)
			DeployApplication(podNamespace, podYaml, namePrefix, "disabled", *t)
		})
		WhenMetricsBindingInstalledIt("Apply the ReplicaSet", func() {
			DeployApplication(replicasetNamespace, replicasetYaml, namePrefix, "enabled", *t)
		})
		WhenMetricsBindingInstalledIt("Apply the StatefulSet", func() {
			DeployApplication(statefulsetNamespace, statefulsetYaml, namePrefix, "disabled", *t)
		})
	})
})
