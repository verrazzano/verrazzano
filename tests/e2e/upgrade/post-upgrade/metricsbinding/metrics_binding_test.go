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

	// Define the test yaml file locations
	deploymentYaml  = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-deployment.yaml"
	podYaml         = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-pod.yaml"
	replicasetYaml  = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-replicaset.yaml"
	statefulsetYaml = "tests/e2e/upgrade/pre-upgrade/metricsbinding/testdata/hello-helidon-statefulset.yaml"
)

var (
	t           = framework.NewTestFramework("deploymentworkload")
	clusterDump = pkg.NewClusterDumpWrapper(deploymentNamespace, podNamespace, replicasetNamespace, statefulsetNamespace)
)

var _ = clusterDump.BeforeSuite(func() {}) // Needed to initialize cluster dump flags
var _ = clusterDump.AfterEach(func() {})   // Dump cluster if spec fails

var _ = clusterDump.AfterSuite(func() {
	undeployApplication(deploymentNamespace, deploymentYaml, *t)
	undeployApplication(podNamespace, podYaml, *t)
	undeployApplication(replicasetNamespace, replicasetYaml, *t)
	undeployApplication(statefulsetNamespace, statefulsetYaml, *t)
})

var _ = t.AfterEach(func() {})

// 'It' Wrapper to only run spec if the Metrics Binding verification is supported
func WhenMetricsBindingInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
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
	// GIVEN the Verrazzano version
	// WHEN the Metrics Binding utilities are updated
	// THEN the Metrics Bindings should be deleted for default template and binding
	t.Context("Verify Metrics Bindings are deleted", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		WhenMetricsBindingInstalledIt("Verify no Metrics Bindings exist in the ReplicaSet namespace", func() {
			verifyMetricsBindingsDeleted(replicasetNamespace, *t)
		})
		WhenMetricsBindingInstalledIt("Verify no Metrics Bindings exist in the StatefulSet namespace", func() {
			verifyMetricsBindingsDeleted(statefulsetNamespace, *t)
		})
	})
})
