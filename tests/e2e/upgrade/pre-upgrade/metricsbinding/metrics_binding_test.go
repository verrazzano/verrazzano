// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
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
	clusterDump = pkg.NewClusterDumpWrapper(deploymentNamespace, podNamespace, replicasetNamespace, statefulsetNamespace)
)

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.AfterEach(func() {})

// 'It' Wrapper to only run spec if the Metrics Binding will be created
func WhenMetricsBindingInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	vz14OrLater, err := pkg.IsVerrazzanoMinVersionEventually("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version less than 1.4.0: %s", err.Error()))
		})
	}
	below14 := !vz14OrLater
	above12, err := pkg.IsVerrazzanoMinVersionEventually("1.2.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version at or above 1.2.0: %s", err.Error()))
		})
	}
	if below14 && above12 {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Metrics Binding is not supported", description)
	}
}

var _ = t.Describe("Verify", Label("f:app-lcm.poko"), func() {
	// GIVEN the Verrazzano version
	// WHEN the sample applications are deployed
	// THEN no errors should occur
	t.Context("Deploy and verify the test applications", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		WhenMetricsBindingInstalledIt("Apply the Deployment and external Prometheus Metrics Template", func() {
			createNamespace(deploymentNamespace, "enabled", *t)
			deployConfigMap(deploymentNamespace, configMapYaml, *t)
			deployTemplate(deploymentNamespace, externalPromTemplate, *t)
			deployApplication(deploymentNamespace, deploymentYaml, namePrefix, *t)
		})
		WhenMetricsBindingInstalledIt("Apply the Pod and legacy VMI Metrics Template", func() {
			createNamespace(podNamespace, "disabled", *t)
			deployTemplate(podNamespace, legacyVMITemplate, *t)
			deployApplication(podNamespace, podYaml, namePrefix, *t)
		})
		WhenMetricsBindingInstalledIt("Apply the ReplicaSet", func() {
			createNamespace(replicasetNamespace, "enabled", *t)
			deployApplication(replicasetNamespace, replicasetYaml, namePrefix, *t)
		})
		WhenMetricsBindingInstalledIt("Apply the StatefulSet", func() {
			createNamespace(statefulsetNamespace, "disabled", *t)
			deployApplication(statefulsetNamespace, statefulsetYaml, namePrefix, *t)
		})
	})
})
